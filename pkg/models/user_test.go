package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMarshalBinary(t *testing.T) {
	userCases, groupCases := getUseCases()

	for _, c := range userCases {
		response, err := c.MarshalBinary()
		if c.expectedString != string(response) {
			t.Errorf("User case: Expected response was not returned.\nExpected: %s\nActual:   %s", c.expectedString, response)
		}
		if err != nil {
			t.Errorf("User case: Did not expect error: %q", err)
		}
	}

	for _, c := range groupCases {
		response, err := c.MarshalBinary()
		if c.expectedString != string(response) {
			t.Errorf("Group case: Expected response was not returned.\nExpected: %s\nActual:   %s", c.expectedString, response)
		}
		if err != nil {
			t.Errorf("Group case: Did not expect error: %q", err)
		}
	}
}

func TestUnmarshalBinary(t *testing.T) {
	userCases, groupCases := getUseCases()

	for _, c := range userCases {
		user := User{}
		err := user.UnmarshalBinary([]byte(c.expectedString))
		if !cmp.Equal(c.User, user) {
			t.Errorf("User case: Expected response was not returned.\nExpected: %s\nActual:   %s", c.User, user)
		}
		if err != nil {
			t.Errorf("User case: Did not expect error: %q", err)
		}
	}

	for _, c := range groupCases {
		group := Group{}
		err := group.UnmarshalBinary([]byte(c.expectedString))
		if !cmp.Equal(c.Group, group) {
			t.Errorf("Group case: Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, group)
		}
		if err != nil {
			t.Errorf("Group case: Did not expect error: %q", err)
		}
	}
}

type userCase struct {
	User
	expectedString string
	expectedErr    error
}

type groupCase struct {
	Group
	expectedString string
}

func getUseCases() ([]userCase, []groupCase) {
	userCases := []userCase{
		{
			User: User{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups:   []Group{},
				Type:     NormalUserType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[],\"Type\":\"NormalUser\"}",
		},
		{
			User: User{
				Username: "00000000-0000-0000-0000-000000000000",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups:   []Group{},
				Type:     ServicePrincipalUserType,
			},
			expectedString: "{\"Username\":\"00000000-0000-0000-0000-000000000000\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[],\"Type\":\"ServicePrincipal\"}",
		},
		{
			User: User{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []Group{
					{Name: "test1"},
				},
				Type: NormalUserType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\"}],\"Type\":\"NormalUser\"}",
		},
		{
			User: User{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []Group{
					{Name: "test1"},
					{Name: "test2"},
				},
				Type: NormalUserType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\"},{\"Name\":\"test2\"}],\"Type\":\"NormalUser\"}",
		},
	}

	groupCases := []groupCase{
		{
			Group: Group{
				Name: "test1",
			},
			expectedString: "{\"Name\":\"test1\"}",
		},
	}

	return userCases, groupCases
}
