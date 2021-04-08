package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	session "github.com/aws/aws-sdk-go/aws/session"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/perchcredit/gqlgenc/graphqljson"
	"github.com/perchcredit/gqlgenc/introspection"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/xerrors"
)

// HTTPRequestOption represents the options applicable to the http client
type HTTPRequestOption func(req *http.Request)

// ----- Client ---------------------------------------------------

// Client is the http client wrapper
type Client struct {
	BaseURL            string
	Client             *http.Client
	HTTPRequestOptions []HTTPRequestOption
	Authorization      ClientAuthorization
}

type ClientAuthorization struct {
	CognitoIdentityProvider *cognito.CognitoIdentityProvider
	ClientID                string
	UserPoolID              string
	Username                string
	Password                string
}

// Request represents an outgoing GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// ----- Client Initialization Options ----------------------------

type ClientOptions struct {
	HTTPClient           *http.Client
	HTTPRequestOptions   []HTTPRequestOption
	BaseURL              string
	AuthorizationOptions ClientAuthorizationOptions
}

type ClientAuthorizationOptions struct {
	Session    *session.Session
	ClientID   string
	UserPoolID string
	Username   string
	Password   string
}

// ----- Client Constructor ----------------------------------------

// NewClient creates a new http client wrapper
func NewClient(options ClientOptions) *Client {
	return &Client{
		Client:             options.HTTPClient,
		HTTPRequestOptions: options.HTTPRequestOptions,
		BaseURL:            options.BaseURL,
		Authorization: ClientAuthorization{
			CognitoIdentityProvider: cognito.New(options.AuthorizationOptions.Session),
			UserPoolID:              options.AuthorizationOptions.UserPoolID,
			ClientID:                options.AuthorizationOptions.ClientID,
			Username:                options.AuthorizationOptions.Username,
			Password:                options.AuthorizationOptions.Password,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, operationName, query string, vars map[string]interface{}, httpRequestOptions []HTTPRequestOption) (*http.Request, error) {

	// Create request object
	// Fill query
	// Fill variables
	r := &Request{
		Query:     query,
		Variables: vars,
	}

	// Marshal request body
	// Exit on error
	requestBody, err := json.Marshal(r)
	if err != nil {
		return nil, xerrors.Errorf("encode: %w", err)
	}

	// Create new request
	// Exit on error
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("create request struct failed: %w", err)
	}

	// If query is not introspection query
	// Add appropriate authorization headers
	if query != introspection.Introspection {

		// Login with cognito admin credentials
		// Exit on error
		login, err := c.Authorization.CognitoIdentityProvider.AdminInitiateAuth(&cognito.AdminInitiateAuthInput{
			AuthFlow:   aws.String("ADMIN_USER_PASSWORD_AUTH"),
			ClientId:   &c.Authorization.ClientID,
			UserPoolId: &c.Authorization.UserPoolID,
			AuthParameters: map[string]*string{
				"USERNAME": aws.String(c.Authorization.Username),
				"PASSWORD": aws.String(c.Authorization.Password),
			},
		})
		if err != nil {
			return nil, xerrors.Errorf("failed to login : %w", err)
		}

		// If authentication result is successful and id token can be parsed
		// Add in authentication header
		if login != nil && login.AuthenticationResult != nil && login.AuthenticationResult.IdToken != nil {
			req.Header.Add("Authorization", "Bearer "+*login.AuthenticationResult.IdToken)
		}
	}

	// Add HTTP Options
	for _, httpRequestOption := range c.HTTPRequestOptions {
		httpRequestOption(req)
	}
	for _, httpRequestOption := range httpRequestOptions {
		httpRequestOption(req)
	}

	return req, nil
}

// GqlErrorList is the struct of a standard graphql error response
type GqlErrorList struct {
	Errors gqlerror.List `json:"errors"`
}

func (e *GqlErrorList) Error() string {
	return e.Errors.Error()
}

// HTTPError is the error when a GqlErrorList cannot be parsed
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represent an handled error
type ErrorResponse struct {
	// populated when http status code is not OK
	NetworkError *HTTPError `json:"networkErrors"`
	// populated when http status code is OK but the server returned at least one graphql error
	GqlErrors *gqlerror.List `json:"graphqlErrors"`
}

// HasErrors returns true when at least one error is declared
func (er *ErrorResponse) HasErrors() bool {
	return er.NetworkError != nil || er.GqlErrors != nil
}

func (er *ErrorResponse) Error() string {
	content, err := json.Marshal(er)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

// Post sends a http POST request to the graphql endpoint with the given query then unpacks
// the response into the given object.
func (c *Client) Post(ctx context.Context, operationName, query string, respData interface{}, vars map[string]interface{}, httpRequestOptions ...HTTPRequestOption) error {
	req, err := c.newRequest(ctx, operationName, query, vars, httpRequestOptions)
	if err != nil {
		return xerrors.Errorf("don't create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	resp, err := c.Client.Do(req)
	if err != nil {
		return xerrors.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return xerrors.Errorf("failed to read response body: %w", err)
	}

	return parseResponse(body, resp.StatusCode, respData)
}

func parseResponse(body []byte, httpCode int, result interface{}) error {
	errResponse := &ErrorResponse{}
	isKOCode := httpCode < 200 || 299 < httpCode
	if isKOCode {
		errResponse.NetworkError = &HTTPError{
			Code:    httpCode,
			Message: fmt.Sprintf("Response body %s", string(body)),
		}
	}

	// some servers return a graphql error with a non OK http code, try anyway to parse the body
	if err := unmarshal(body, result); err != nil {
		if gqlErr, ok := err.(*GqlErrorList); ok {
			errResponse.GqlErrors = &gqlErr.Errors
		} else if !isKOCode { // if is KO code there is already the http error, this error should not be returned
			return err
		}
	}

	if errResponse.HasErrors() {
		return errResponse
	}

	return nil
}

// response is a GraphQL layer response from a handler.
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

func unmarshal(data []byte, res interface{}) error {
	resp := response{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return xerrors.Errorf("failed to decode data %s: %w", string(data), err)
	}

	if resp.Errors != nil && len(resp.Errors) > 0 {
		// try to parse standard graphql error
		errors := &GqlErrorList{}
		if e := json.Unmarshal(data, errors); e != nil {
			return xerrors.Errorf("faild to parse graphql errors. Response content %s - %w ", string(data), e)
		}

		return errors
	}

	if err := graphqljson.UnmarshalData(resp.Data, res); err != nil {
		return xerrors.Errorf("failed to decode data into response %s: %w", string(data), err)
	}

	return nil
}
