//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2024 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package test

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/client/authz"
	"github.com/weaviate/weaviate/client/objects"
	clschema "github.com/weaviate/weaviate/client/schema"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/schema"
	"github.com/weaviate/weaviate/test/helper"
	"github.com/weaviate/weaviate/usecases/auth/authorization"
)

func TestAuthZObjectValidate(t *testing.T) {
	adminKey := "admin-key"
	adminAuth := helper.CreateAuth(adminKey)
	customUser := "custom-user"
	customAuth := helper.CreateAuth("custom-key")

	readDataAction := authorization.ReadObjectsCollection
	readSchemaAction := authorization.ReadCollections

	helper.SetupClient("127.0.0.1:8081")
	defer helper.ResetClient()

	roleName := "AuthZObjectValidateTestRole"
	className := "AuthZObjectValidateTest"
	c := &models.Class{
		Class: className,
		Properties: []*models.Property{
			{
				Name:     "prop",
				DataType: schema.DataTypeText.PropString(),
			},
		},
	}
	deleteObjectClass(t, className, adminAuth)
	params := clschema.NewSchemaObjectsCreateParams().WithObjectClass(c)
	resp, err := helper.Client(t).Schema.SchemaObjectsCreate(params, adminAuth)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	t.Run("All rights", func(t *testing.T) {
		deleteRole := &models.Role{
			Name: &roleName,
			Permissions: []*models.Permission{
				{
					Action:     &readDataAction,
					Collection: &className,
				},
				{
					Action:     &readSchemaAction,
					Collection: &className,
				},
			},
		}
		helper.DeleteRole(t, adminKey, *deleteRole.Name)
		helper.CreateRole(t, adminKey, deleteRole)
		_, err = helper.Client(t).Authz.AssignRole(
			authz.NewAssignRoleParams().WithID(customUser).WithBody(authz.AssignRoleBody{Roles: []string{roleName}}),
			adminAuth,
		)
		require.Nil(t, err)

		paramsObj := objects.NewObjectsValidateParams().WithBody(
			&models.Object{
				ID:    strfmt.UUID(uuid.New().String()),
				Class: className,
				Properties: map[string]interface{}{
					"prop": "test",
				},
			})
		_, err := helper.Client(t).Objects.ObjectsValidate(paramsObj, customAuth)
		require.Nil(t, err)

		_, err = helper.Client(t).Authz.RevokeRole(
			authz.NewRevokeRoleParams().WithID(customUser).WithBody(authz.RevokeRoleBody{Roles: []string{roleName}}),
			adminAuth,
		)
		require.Nil(t, err)
		helper.DeleteRole(t, adminKey, roleName)
	})

	actions := []string{readDataAction, readSchemaAction}
	for _, action := range actions {
		t.Run("Only rights for "+action, func(t *testing.T) {
			deleteRole := &models.Role{
				Name: &roleName,
				Permissions: []*models.Permission{{
					Action:     &action,
					Collection: &className,
				}},
			}
			helper.DeleteRole(t, adminKey, *deleteRole.Name)
			helper.CreateRole(t, adminKey, deleteRole)
			_, err = helper.Client(t).Authz.AssignRole(
				authz.NewAssignRoleParams().WithID(customUser).WithBody(authz.AssignRoleBody{Roles: []string{roleName}}),
				adminAuth,
			)
			require.Nil(t, err)

			paramsObj := objects.NewObjectsValidateParams().WithBody(
				&models.Object{
					ID:    strfmt.UUID(uuid.New().String()),
					Class: className,
					Properties: map[string]interface{}{
						"prop": "test",
					},
				})
			_, err := helper.Client(t).Objects.ObjectsValidate(paramsObj, customAuth)
			require.NotNil(t, err)
			var errNoAuth *objects.ObjectsValidateForbidden
			require.True(t, errors.As(err, &errNoAuth))

			_, err = helper.Client(t).Authz.RevokeRole(
				authz.NewRevokeRoleParams().WithID(customUser).WithBody(authz.RevokeRoleBody{Roles: []string{roleName}}),
				adminAuth,
			)
			require.Nil(t, err)
			helper.DeleteRole(t, adminKey, roleName)
		})
	}
}
