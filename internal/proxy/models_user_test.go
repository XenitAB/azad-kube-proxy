package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalBinary(t *testing.T) {
	testUserCases, testGroupCases := testGetUseCases(t)

	for _, c := range testUserCases {
		response, err := c.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, c.expectedString, string(response))
	}

	for _, c := range testGroupCases {
		response, err := c.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, c.expectedString, string(response))
	}
}

func TestUnmarshalBinary(t *testing.T) {
	testUserCases, testGroupCases := testGetUseCases(t)

	for _, c := range testUserCases {
		user := userModel{}
		err := user.UnmarshalBinary([]byte(c.expectedString))
		require.NoError(t, err)
		require.Equal(t, c.userModel, user)
	}

	for _, c := range testGroupCases {
		group := groupModel{}
		err := group.UnmarshalBinary([]byte(c.expectedString))
		require.NoError(t, err)
		require.Equal(t, c.groupModel, group)
	}
}

func TestGetGroupIdentifier(t *testing.T) {
	cases := []struct {
		groupIdentifierString   string
		expectedGroupIdentifier groupIdentifier
		expectedErrContains     string
	}{
		{
			groupIdentifierString:   "NAME",
			expectedGroupIdentifier: nameGroupIdentifier,
			expectedErrContains:     "",
		},
		{
			groupIdentifierString:   "OBJECTID",
			expectedGroupIdentifier: objectIDGroupIdentifier,
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

type testUserCase struct {
	userModel
	expectedString string
}

type testGroupCase struct {
	groupModel
	expectedString string
}

func testGetUseCases(t *testing.T) ([]testUserCase, []testGroupCase) {
	t.Helper()

	testUserCases := []testUserCase{
		{
			userModel: userModel{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups:   []groupModel{},
				Type:     normalUserModelType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[],\"Type\":\"NormalUser\"}",
		},
		{
			userModel: userModel{
				Username: "00000000-0000-0000-0000-000000000000",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups:   []groupModel{},
				Type:     servicePrincipalUserModelType,
			},
			expectedString: "{\"Username\":\"00000000-0000-0000-0000-000000000000\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[],\"Type\":\"ServicePrincipal\"}",
		},
		{
			userModel: userModel{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []groupModel{
					{
						Name:     "test1",
						ObjectID: "00000000-0000-0000-0000-000000000000",
					},
				},
				Type: normalUserModelType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"}],\"Type\":\"NormalUser\"}",
		},
		{
			userModel: userModel{
				Username: "username",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []groupModel{
					{
						Name:     "test1",
						ObjectID: "00000000-0000-0000-0000-000000000000",
					},
					{
						Name:     "test2",
						ObjectID: "00000000-0000-0000-0000-000000000001",
					},
				},
				Type: normalUserModelType,
			},
			expectedString: "{\"Username\":\"username\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\",\"Groups\":[{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"},{\"Name\":\"test2\",\"ObjectID\":\"00000000-0000-0000-0000-000000000001\"}],\"Type\":\"NormalUser\"}",
		},
	}

	testGroupCases := []testGroupCase{
		{
			groupModel: groupModel{
				Name:     "test1",
				ObjectID: "00000000-0000-0000-0000-000000000000",
			},
			expectedString: "{\"Name\":\"test1\",\"ObjectID\":\"00000000-0000-0000-0000-000000000000\"}",
		},
	}

	return testUserCases, testGroupCases
}
