package repository

import (
	"context"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}

// setupTestDB creates a test database connection using SQLite in-memory database.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate test model
	err = db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	return db
}

// TestGormRepository_Create tests the Create method.
func TestGormRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	t.Run("create_success", func(t *testing.T) {
		user := &model.User{
			Name:  "John",
			Email: stringPtr("john@example.com"),
			Phone: stringPtr("13800138001"),
		}

		err := repo.Create(ctx, user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
		assert.Equal(t, model.UserStatusNormal, user.Status)
	})

	t.Run("create_with_duplicate_email", func(t *testing.T) {
		user1 := &model.User{
			Name:  "John",
			Email: stringPtr("duplicate@example.com"),
			Phone: stringPtr("13800138002"),
		}
		err := repo.Create(ctx, user1)
		assert.NoError(t, err)

		// Try to create another user with same email (unique constraint)
		user2 := &model.User{
			Name:  "Jane",
			Email: stringPtr("duplicate@example.com"),
			Phone: stringPtr("13800138003"),
		}
		err = repo.Create(ctx, user2)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

// TestGormRepository_Update tests the Update method.
func TestGormRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create a user first
	user := &model.User{
		Name:  "John",
		Email: stringPtr("john@example.com"),
		Phone: stringPtr("13800138004"),
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	originalUpdatedAt := user.UpdatedAt

	t.Run("update_success", func(t *testing.T) {
		user.Name = "Jane"
		time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt changes

		err := repo.Update(ctx, user)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", updated.Name)
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("update_with_duplicate_email", func(t *testing.T) {
		// Create another user with different email
		user2 := &model.User{
			Name:  "Bob",
			Email: stringPtr("bob@example.com"),
			Phone: stringPtr("13800138004_2"),
		}
		err := repo.Create(ctx, user2)
		require.NoError(t, err)

		// Try to update user2 with duplicate email
		user2.Email = user.Email // Set to existing email
		err = repo.Update(ctx, user2)
		assert.Error(t, err) // Should fail due to unique constraint
	})

	t.Run("update_with_duplicate_phone", func(t *testing.T) {
		// Create another user with different phone
		user3 := &model.User{
			Name:  "Alice",
			Email: stringPtr("alice@example.com"),
			Phone: stringPtr("13800138004_3"),
		}
		err := repo.Create(ctx, user3)
		require.NoError(t, err)

		// Try to update user3 with duplicate phone
		user3.Phone = user.Phone // Set to existing phone
		err = repo.Update(ctx, user3)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

// TestGormRepository_Delete tests the Delete method.
func TestGormRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	t.Run("delete_success", func(t *testing.T) {
		// Create a user
		user := &model.User{
			Name:  "John",
			Email: stringPtr("john@example.com"),
			Phone: stringPtr("13800138005"),
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
		id := user.ID

		// Delete the user
		err = repo.Delete(ctx, id)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repo.FindByID(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete_not_found", func(t *testing.T) {
		err := repo.Delete(ctx, 99999)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete_with_invalid_id", func(t *testing.T) {
		// Test with ID 0 (invalid)
		err := repo.Delete(ctx, 0)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete_multiple_times", func(t *testing.T) {
		// Create a user
		user := &model.User{
			Name:  "TestDelete",
			Email: stringPtr("testdelete@example.com"),
			Phone: stringPtr("13800138006"),
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
		id := user.ID

		// Delete the user first time
		err = repo.Delete(ctx, id)
		assert.NoError(t, err)

		// Try to delete again (should fail with ErrNotFound)
		err = repo.Delete(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

// TestGormRepository_FindByID tests the FindByID method.
func TestGormRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	t.Run("find_success", func(t *testing.T) {
		// Create a user
		user := &model.User{
			Name:  "John",
			Email: stringPtr("john@example.com"),
			Phone: stringPtr("13800138006"),
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Find by ID
		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, "John", found.Name)
		assert.Equal(t, "john@example.com", found.GetEmail())
	})

	t.Run("find_not_found", func(t *testing.T) {
		_, err := repo.FindByID(ctx, 99999)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

// TestGormRepository_FindByIDs tests the FindByIDs method.
func TestGormRepository_FindByIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create multiple users
	user1 := &model.User{Name: "John", Email: stringPtr("john@example.com"), Phone: stringPtr("13800138007")}
	user2 := &model.User{Name: "Jane", Email: stringPtr("jane@example.com"), Phone: stringPtr("13800138008")}
	user3 := &model.User{Name: "Bob", Email: stringPtr("bob@example.com"), Phone: stringPtr("13800138009")}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)
	err = repo.Create(ctx, user3)
	require.NoError(t, err)

	t.Run("find_multiple_ids", func(t *testing.T) {
		ids := []uint{user1.ID, user3.ID}
		users, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, user1.ID, users[0].ID)
		assert.Equal(t, user3.ID, users[1].ID)
	})

	t.Run("find_empty_ids", func(t *testing.T) {
		users, err := repo.FindByIDs(ctx, []uint{})
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("find_nonexistent_ids", func(t *testing.T) {
		ids := []uint{99999, 99998}
		users, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})
}

// TestGormRepository_CreateBatch tests the CreateBatch method.
func TestGormRepository_CreateBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	t.Run("create_batch_success", func(t *testing.T) {
		users := []*model.User{
			{Name: "User1", Email: stringPtr("user1@example.com"), Phone: stringPtr("13800138010")},
			{Name: "User2", Email: stringPtr("user2@example.com"), Phone: stringPtr("13800138011")},
			{Name: "User3", Email: stringPtr("user3@example.com"), Phone: stringPtr("13800138012")},
		}

		err := repo.CreateBatch(ctx, users)
		assert.NoError(t, err)

		// Verify all users were created
		assert.NotZero(t, users[0].ID)
		assert.NotZero(t, users[1].ID)
		assert.NotZero(t, users[2].ID)

		// Verify in database
		allUsers, err := repo.FindWhereMap(ctx, nil, "")
		assert.NoError(t, err)
		assert.Len(t, allUsers, 3)
	})

	t.Run("create_batch_empty", func(t *testing.T) {
		err := repo.CreateBatch(ctx, []*model.User{})
		assert.NoError(t, err)
	})

	t.Run("create_batch_with_duplicate_email", func(t *testing.T) {
		// Create a user first
		existingUser := &model.User{
			Name:  "Existing",
			Email: stringPtr("duplicate@example.com"),
			Phone: stringPtr("13800138013"),
		}
		err := repo.Create(ctx, existingUser)
		require.NoError(t, err)

		// Try to create batch with duplicate email
		users := []*model.User{
			{Name: "User1", Email: stringPtr("duplicate@example.com"), Phone: stringPtr("13800138014")},
		}

		err = repo.CreateBatch(ctx, users)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

// TestGormRepository_UpdateByID tests the UpdateByID method.
func TestGormRepository_UpdateByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create a user
	user := &model.User{
		Name:  "John",
		Email: stringPtr("john@example.com"),
		Phone: stringPtr("13800138013"),
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	t.Run("update_by_id_success", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Jane",
		}

		err := repo.UpdateByID(ctx, user.ID, updates)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", updated.Name)
		assert.Equal(t, "john@example.com", updated.GetEmail()) // Email should remain unchanged
	})

	t.Run("update_by_id_empty_updates", func(t *testing.T) {
		err := repo.UpdateByID(ctx, user.ID, map[string]interface{}{})
		assert.NoError(t, err) // Should return nil without error
	})

	t.Run("update_by_id_with_duplicate_email", func(t *testing.T) {
		// Create another user
		user2 := &model.User{
			Name:  "Bob",
			Email: stringPtr("bob@example.com"),
			Phone: stringPtr("13800138014"),
		}
		err := repo.Create(ctx, user2)
		require.NoError(t, err)

		// Try to update user2 with duplicate email
		updates := map[string]interface{}{
			"email": user.Email,
		}

		err = repo.UpdateByID(ctx, user2.ID, updates)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

// TestGormRepository_UpdateWhere tests the UpdateWhere method.
func TestGormRepository_UpdateWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create users
	user1 := &model.User{Name: "John", Email: stringPtr("john@example.com"), Phone: stringPtr("13800138014")}
	user2 := &model.User{Name: "John", Email: stringPtr("john2@example.com"), Phone: stringPtr("13800138015")}
	user3 := &model.User{Name: "Jane", Email: stringPtr("jane@example.com"), Phone: stringPtr("13800138016")}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)
	err = repo.Create(ctx, user3)
	require.NoError(t, err)

	t.Run("update_where_success", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "John",
		}
		updates := map[string]interface{}{
			"name": "Johnny",
		}

		err := repo.UpdateWhere(ctx, conditions, updates)
		assert.NoError(t, err)

		// Verify updates
		allUsers, err := repo.FindWhereMap(ctx, map[string]interface{}{"name": "Johnny"}, "")
		assert.NoError(t, err)
		assert.Len(t, allUsers, 2) // Both John users should be updated
	})

	t.Run("update_where_empty_conditions", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "ShouldNotUpdate",
		}

		err := repo.UpdateWhere(ctx, nil, updates)
		assert.NoError(t, err) // Should return nil without updating

		// Verify no updates occurred
		users, err := repo.FindWhereMap(ctx, map[string]interface{}{"name": "ShouldNotUpdate"}, "")
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("update_where_empty_updates", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "John",
		}

		err := repo.UpdateWhere(ctx, conditions, map[string]interface{}{})
		assert.NoError(t, err) // Should return nil without error
	})

	t.Run("update_where_with_duplicate_email", func(t *testing.T) {
		// Create another user
		user4 := &model.User{
			Name:  "Charlie",
			Email: stringPtr("charlie@example.com"),
			Phone: stringPtr("13800138020"),
		}
		err := repo.Create(ctx, user4)
		require.NoError(t, err)

		// Try to update user4 with duplicate email
		conditions := map[string]interface{}{
			"name": "Charlie",
		}
		updates := map[string]interface{}{
			"email": user1.Email, // Use existing user's email
		}

		err = repo.UpdateWhere(ctx, conditions, updates)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

// TestGormRepository_DeleteWhere tests the DeleteWhere method.
func TestGormRepository_DeleteWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create users
	user1 := &model.User{Name: "John", Email: stringPtr("john@example.com"), Phone: stringPtr("13800138017")}
	user2 := &model.User{Name: "John", Email: stringPtr("john2@example.com"), Phone: stringPtr("13800138018")}
	user3 := &model.User{Name: "Jane", Email: stringPtr("jane@example.com"), Phone: stringPtr("13800138019")}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)
	err = repo.Create(ctx, user3)
	require.NoError(t, err)

	t.Run("delete_where_success", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "John",
		}

		err := repo.DeleteWhere(ctx, conditions)
		assert.NoError(t, err)

		// Verify deletion
		remaining, err := repo.FindWhereMap(ctx, map[string]interface{}{"name": "John"}, "")
		assert.NoError(t, err)
		assert.Len(t, remaining, 0)

		// Verify Jane still exists
		jane, err := repo.FindWhereMap(ctx, map[string]interface{}{"name": "Jane"}, "")
		assert.NoError(t, err)
		assert.Len(t, jane, 1)
	})

	t.Run("delete_where_empty_conditions", func(t *testing.T) {
		// Create a user first
		user := &model.User{Name: "Test", Email: stringPtr("test@example.com"), Phone: stringPtr("13800138020")}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Try to delete with empty conditions
		err = repo.DeleteWhere(ctx, nil)
		assert.NoError(t, err) // Should return nil without deleting

		// Verify user still exists
		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// TestGormRepository_FindWhereMap tests the FindWhereMap method.
func TestGormRepository_FindWhereMap(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create users
	user1 := &model.User{Name: "John", Email: stringPtr("john@example.com"), Phone: stringPtr("13800138021")}
	user2 := &model.User{Name: "Jane", Email: stringPtr("jane@example.com"), Phone: stringPtr("13800138022")}
	user3 := &model.User{Name: "Bob", Email: stringPtr("bob@example.com"), Phone: stringPtr("13800138023")}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)
	err = repo.Create(ctx, user3)
	require.NoError(t, err)

	t.Run("find_with_conditions", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "John",
		}

		users, err := repo.FindWhereMap(ctx, conditions, "")
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "John", users[0].Name)
	})

	t.Run("find_without_conditions", func(t *testing.T) {
		users, err := repo.FindWhereMap(ctx, nil, "")
		assert.NoError(t, err)
		assert.Len(t, users, 3)
	})

	t.Run("find_with_order", func(t *testing.T) {
		users, err := repo.FindWhereMap(ctx, nil, "name ASC")
		assert.NoError(t, err)
		assert.Len(t, users, 3)
		// Verify ordering (Bob, Jane, John)
		assert.Equal(t, "Bob", users[0].Name)
		assert.Equal(t, "Jane", users[1].Name)
		assert.Equal(t, "John", users[2].Name)
	})

	t.Run("find_with_desc_order", func(t *testing.T) {
		users, err := repo.FindWhereMap(ctx, nil, "name DESC")
		assert.NoError(t, err)
		assert.Len(t, users, 3)
		// Verify ordering (John, Jane, Bob)
		assert.Equal(t, "John", users[0].Name)
		assert.Equal(t, "Jane", users[1].Name)
		assert.Equal(t, "Bob", users[2].Name)
	})
}

// TestGormRepository_FindPage tests the FindPage method.
func TestGormRepository_FindPage(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create multiple users
	users := []*model.User{
		{Name: "User1", Email: stringPtr("user1@example.com"), Phone: stringPtr("13800138024")},
		{Name: "User2", Email: stringPtr("user2@example.com"), Phone: stringPtr("13800138025")},
		{Name: "User3", Email: stringPtr("user3@example.com"), Phone: stringPtr("13800138026")},
		{Name: "User4", Email: stringPtr("user4@example.com"), Phone: stringPtr("13800138027")},
		{Name: "User5", Email: stringPtr("user5@example.com"), Phone: stringPtr("13800138028")},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	t.Run("find_page_first_page", func(t *testing.T) {
		records, total, err := repo.FindPage(ctx, nil, 1, 2, "name ASC")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, records, 2)
		assert.Equal(t, "User1", records[0].Name)
		assert.Equal(t, "User2", records[1].Name)
	})

	t.Run("find_page_second_page", func(t *testing.T) {
		records, total, err := repo.FindPage(ctx, nil, 2, 2, "name ASC")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, records, 2)
		assert.Equal(t, "User3", records[0].Name)
		assert.Equal(t, "User4", records[1].Name)
	})

	t.Run("find_page_with_conditions", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "User1",
		}

		records, total, err := repo.FindPage(ctx, conditions, 1, 10, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, records, 1)
		assert.Equal(t, "User1", records[0].Name)
	})

	t.Run("find_page_defaults", func(t *testing.T) {
		// Test default page (should be 1) and default pageSize (should be 10)
		// But since we only have 5 records, it should return all 5
		records, total, err := repo.FindPage(ctx, nil, 0, 0, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, records, 5) // Default pageSize is 10, but only 5 records exist
	})

	t.Run("find_page_empty_result", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "Nonexistent",
		}

		records, total, err := repo.FindPage(ctx, conditions, 1, 10, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, records, 0)
	})
}

// TestGormRepository_CountWhere tests the CountWhere method.
func TestGormRepository_CountWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[model.User](db)
	ctx := context.Background()

	// Create users
	user1 := &model.User{Name: "John", Email: stringPtr("john@example.com"), Phone: stringPtr("13800138029")}
	user2 := &model.User{Name: "John", Email: stringPtr("john2@example.com"), Phone: stringPtr("13800138030")}
	user3 := &model.User{Name: "Jane", Email: stringPtr("jane@example.com"), Phone: stringPtr("13800138031")}
	user4 := &model.User{Name: "Bob", Email: stringPtr("bob@example.com"), Phone: stringPtr("13800138032")}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)
	err = repo.Create(ctx, user3)
	require.NoError(t, err)
	err = repo.Create(ctx, user4)
	require.NoError(t, err)

	t.Run("count_all", func(t *testing.T) {
		count, err := repo.CountWhere(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})

	t.Run("count_with_conditions", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "John",
		}

		count, err := repo.CountWhere(ctx, conditions)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count) // Two users named John
	})

	t.Run("count_with_multiple_conditions", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		}

		count, err := repo.CountWhere(ctx, conditions)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count) // Only one user matches both conditions
	})

	t.Run("count_with_no_match", func(t *testing.T) {
		conditions := map[string]interface{}{
			"name": "Nonexistent",
		}

		count, err := repo.CountWhere(ctx, conditions)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("count_with_empty_conditions", func(t *testing.T) {
		count, err := repo.CountWhere(ctx, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Equal(t, int64(4), count) // Should count all records
	})
}
