// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"sync"
)

// RemarkClient is a mock implementation of openai.remarkCommentsGetter.
//
//	func TestSomethingThatUsesremarkCommentsGetter(t *testing.T) {
//
//		// make and configure a mocked openai.remarkCommentsGetter
//		mockedremarkCommentsGetter := &RemarkClient{
//			GetTopCommentsFunc: func(remarkLink string) ([]string, []string, error) {
//				panic("mock out the GetTopComments method")
//			},
//		}
//
//		// use mockedremarkCommentsGetter in code that requires openai.remarkCommentsGetter
//		// and then make assertions.
//
//	}
type RemarkClient struct {
	// GetTopCommentsFunc mocks the GetTopComments method.
	GetTopCommentsFunc func(remarkLink string) ([]string, []string, error)

	// calls tracks calls to the methods.
	calls struct {
		// GetTopComments holds details about calls to the GetTopComments method.
		GetTopComments []struct {
			// RemarkLink is the remarkLink argument value.
			RemarkLink string
		}
	}
	lockGetTopComments sync.RWMutex
}

// GetTopComments calls GetTopCommentsFunc.
func (mock *RemarkClient) GetTopComments(remarkLink string) ([]string, []string, error) {
	if mock.GetTopCommentsFunc == nil {
		panic("RemarkClient.GetTopCommentsFunc: method is nil but remarkCommentsGetter.GetTopComments was just called")
	}
	callInfo := struct {
		RemarkLink string
	}{
		RemarkLink: remarkLink,
	}
	mock.lockGetTopComments.Lock()
	mock.calls.GetTopComments = append(mock.calls.GetTopComments, callInfo)
	mock.lockGetTopComments.Unlock()
	return mock.GetTopCommentsFunc(remarkLink)
}

// GetTopCommentsCalls gets all the calls that were made to GetTopComments.
// Check the length with:
//
//	len(mockedremarkCommentsGetter.GetTopCommentsCalls())
func (mock *RemarkClient) GetTopCommentsCalls() []struct {
	RemarkLink string
} {
	var calls []struct {
		RemarkLink string
	}
	mock.lockGetTopComments.RLock()
	calls = mock.calls.GetTopComments
	mock.lockGetTopComments.RUnlock()
	return calls
}
