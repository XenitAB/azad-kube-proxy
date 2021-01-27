package models

import (
	"errors"
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
					{
						Name:     "test1",
						ObjectID: "00000000-0000-0000-0000-000000000000",
					},
				},
				Type: NormalUserType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"}],\"Type\":\"NormalUser\"}",
		},
		{
			User: User{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []Group{
					{
						Name:     "test1",
						ObjectID: "00000000-0000-0000-0000-000000000000",
					},
					{
						Name:     "test2",
						ObjectID: "00000000-0000-0000-0000-000000000001",
					},
				},
				Type: NormalUserType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"},{\"Name\":\"test2\",\"ObjectID\":\"00000000-0000-0000-0000-000000000001\"}],\"Type\":\"NormalUser\"}",
		},
	}

	groupCases := []groupCase{
		{
			Group: Group{
				Name:     "test1",
				ObjectID: "00000000-0000-0000-0000-000000000000",
			},
			expectedString: "{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"}",
		},
	}

	return userCases, groupCases
}

func TestGetGroupIdentifier(t *testing.T) {
	cases := []struct {
		groupIdentifierString   string
		expectedGroupIdentifier GroupIdentifier
		expectedErr             error
	}{
		{
			groupIdentifierString:   "NAME",
			expectedGroupIdentifier: NameGroupIdentifier,
			expectedErr:             nil,
		},
		{
			groupIdentifierString:   "OBJECTID",
			expectedGroupIdentifier: ObjectIDGroupIdentifier,
			expectedErr:             nil,
		},
		{
			groupIdentifierString:   "",
			expectedGroupIdentifier: "",
			expectedErr:             errors.New("Unknown group identifier . Supported identifiers are: NAME or OBJECTID"),
		},
		{
			groupIdentifierString:   "DUMMY",
			expectedGroupIdentifier: "",
			expectedErr:             errors.New("Unknown group identifier DUMMY. Supported identifiers are: NAME or OBJECTID"),
		},
	}

	for _, c := range cases {
		resGroupIdentifier, err := GetGroupIdentifier(c.groupIdentifierString)

		if resGroupIdentifier != c.expectedGroupIdentifier && c.expectedErr == nil {
			t.Errorf("Expected group identifier (%s) was not returned: %s", c.expectedGroupIdentifier, resGroupIdentifier)
		}

		if err != nil && c.expectedErr == nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if c.expectedErr != nil {
			if err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected err to be %q but it was %q", c.expectedErr, err)
			}
		}
	}
}
