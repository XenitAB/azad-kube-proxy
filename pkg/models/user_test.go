package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalBinary(t *testing.T) {
	userCases, groupCases := getUseCases()

	for _, c := range userCases {
		response, err := c.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, c.expectedString, string(response))
	}

	for _, c := range groupCases {
		response, err := c.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, c.expectedString, string(response))
	}
}

func TestUnmarshalBinary(t *testing.T) {
	userCases, groupCases := getUseCases()

	for _, c := range userCases {
		user := User{}
		err := user.UnmarshalBinary([]byte(c.expectedString))
		require.NoError(t, err)
		require.Equal(t, c.User, user)
	}

	for _, c := range groupCases {
		group := Group{}
		err := group.UnmarshalBinary([]byte(c.expectedString))
		require.NoError(t, err)
		require.Equal(t, c.Group, group)
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
		expectedErrContains     string
	}{
		{
			groupIdentifierString:   "NAME",
			expectedGroupIdentifier: NameGroupIdentifier,
			expectedErrContains:     "",
		},
		{
			groupIdentifierString:   "OBJECTID",
			expectedGroupIdentifier: ObjectIDGroupIdentifier,
			expectedErrContains:     "",
		},
		{
			groupIdentifierString:   "",
			expectedGroupIdentifier: "",
			expectedErrContains:     "Unknown group identifier ''. Supported identifiers are: NAME or OBJECTID",
		},
		{
			groupIdentifierString:   "DUMMY",
			expectedGroupIdentifier: "",
			expectedErrContains:     "Unknown group identifier 'DUMMY'. Supported identifiers are: NAME or OBJECTID",
		},
	}

	for _, c := range cases {
		resGroupIdentifier, err := GetGroupIdentifier(c.groupIdentifierString)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedGroupIdentifier, resGroupIdentifier)
	}
}
