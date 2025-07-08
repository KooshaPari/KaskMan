# Activity Logging System - Implementation Guide

This document describes the comprehensive activity logging system implemented for the KaskMan R&D Platform.

## Overview

The activity logging system captures all user actions and system events for comprehensive audit trails. It includes:

- **User Authentication Activities**: Login, logout, registration attempts
- **CRUD Operations**: Create, read, update, delete operations on resources
- **System Events**: System startup, shutdown, errors, maintenance operations
- **API Request Logging**: Automatic HTTP request logging with performance metrics
- **Advanced Filtering and Search**: Query activities by user, action, resource, date range, etc.

## System Components

### 1. Activity Service (`internal/activity/service.go`)

The core service that provides activity logging functionality:

```go
// Create activity service
activityRepo := repositories.NewActivityLogRepository(db.DB)
activityService := activity.NewService(activityRepo, logger)

// Log user authentication
activityService.LogAuthActivity(ctx, &userID, username, activity.ActivityTypeLogin, true, ipAddress, userAgent, "")

// Log CRUD operations with change tracking
changes := map[string]interface{}{
    "name": map[string]interface{}{"from": "Old Name", "to": "New Name"},
    "status": map[string]interface{}{"from": "draft", "to": "active"},
}
activityService.LogCRUDActivity(ctx, &userID, username, activity.ActivityTypeProjectUpdate, activity.ResourceTypeProject, &resourceID, changes, true, ipAddress, userAgent, "")

// Log system events
activityService.LogSystemActivity(ctx, activity.ActivityTypeSystemBackup, details, true, "")
```

### 2. Activity Middleware (`internal/api/middleware/middleware.go`)

Automatic HTTP request logging middleware:

```go
// Add to router
router.Use(middleware.ActivityLogger(activityService))

// Or with custom options
options := middleware.DefaultActivityLoggerOptions()
router.Use(middleware.ActivityLoggerWithOptions(activityService, options))
```

### 3. Activity Repository (`internal/database/repositories/activity_log_repository.go`)

Database layer for storing and retrieving activity logs with advanced querying capabilities.

### 4. Activity Handlers (`internal/api/handlers/handlers.go`)

API endpoints for querying activity logs:
- `GET /api/v1/activities` - Get activities with filtering and pagination
- `GET /api/v1/activities/recent` - Get recent activities

## Activity Types

### Authentication Activities
- `login` - User login attempts
- `logout` - User logout
- `register` - User registration attempts

### CRUD Activities
- `create` - Resource creation
- `read` - Resource access
- `update` - Resource modification
- `delete` - Resource deletion

### Specific Resource Activities
- `project_create`, `project_update`, `project_delete`, `project_view`
- `task_create`, `task_update`, `task_delete`, `task_assign`, `task_complete`
- `proposal_create`, `proposal_update`, `proposal_delete`, `proposal_approve`, `proposal_reject`
- `agent_create`, `agent_update`, `agent_delete`

### System Activities
- `system_start`, `system_shutdown`, `system_error`
- `system_backup`, `system_restore`
- `api_request`, `api_response`, `api_error`

## Resource Types

- `user` - User-related activities
- `project` - Project operations
- `task` - Task management
- `proposal` - Proposal workflow
- `agent` - AI agent operations
- `system` - System-level events
- `api` - API interactions

## Activity Details Structure

Each activity log contains detailed information:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "action": "project_update",
  "resource": "project",
  "resource_id": "550e8400-e29b-41d4-a716-446655440002",
  "details": {
    "user_agent": "Mozilla/5.0...",
    "ip_address": "192.168.1.100",
    "method": "PUT",
    "path": "/api/v1/projects/550e8400-e29b-41d4-a716-446655440002",
    "status_code": 200,
    "duration_ms": 245,
    "changes": {
      "name": {"from": "Old Project", "to": "New Project"},
      "status": {"from": "draft", "to": "active"}
    },
    "old_values": {
      "name": "Old Project",
      "status": "draft"
    },
    "new_values": {
      "name": "New Project", 
      "status": "active"
    }
  },
  "success": true,
  "error_message": null,
  "created_at": "2024-01-15T10:30:00Z"
}
```

## API Usage Examples

### Get Activities with Filtering

```bash
# Get all activities
GET /api/v1/activities

# Filter by user
GET /api/v1/activities?user_id=550e8400-e29b-41d4-a716-446655440001

# Filter by action
GET /api/v1/activities?action=login

# Filter by resource type
GET /api/v1/activities?resource=project

# Filter by date range
GET /api/v1/activities?start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z

# Search activities
GET /api/v1/activities?search=login%20failed

# Pagination
GET /api/v1/activities?page=2&page_size=25&sort=created_at&order=desc
```

### Get Recent Activities

```bash
# Get 50 most recent activities
GET /api/v1/activities/recent

# Get 10 most recent activities
GET /api/v1/activities/recent?limit=10
```

### Response Format

```json
{
  "activities": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "action": "login",
      "resource": "user",
      "success": true,
      "ip_address": "192.168.1.100",
      "created_at": "2024-01-15T10:30:00Z",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "username": "john.doe",
        "email": "john.doe@example.com"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 1234,
    "total_pages": 25
  }
}
```

## Integration in Handlers

### Authentication Handlers

```go
// In Login handler
tokens, user, err := h.authService.Login(&req)
if err != nil {
    // Log failed login
    h.activityService.LogAuthActivity(
        c.Request.Context(),
        nil,
        req.Username,
        activity.ActivityTypeLogin,
        false,
        c.ClientIP(),
        c.Request.UserAgent(),
        err.Error(),
    )
    // ... handle error
    return
}

// Log successful login
h.activityService.LogAuthActivity(
    c.Request.Context(),
    &user.ID,
    user.Username,
    activity.ActivityTypeLogin,
    true,
    c.ClientIP(),
    c.Request.UserAgent(),
    "",
)
```

### CRUD Operations with Change Tracking

```go
// In UpdateProject handler
changes := make(map[string]interface{})
oldValues := make(map[string]interface{})
newValues := make(map[string]interface{})

// Track changes
if req.Name != nil && project.Name != *req.Name {
    oldValues["name"] = project.Name
    newValues["name"] = *req.Name
    changes["name"] = map[string]interface{}{"from": project.Name, "to": *req.Name}
    project.Name = *req.Name
}

// Save changes
if err := h.db.DB.Save(&project).Error; err != nil {
    // Log failed update
    userID, username := h.activityService.ExtractUserInfoFromContext(c)
    h.activityService.LogCRUDActivity(
        c.Request.Context(),
        userID,
        username,
        activity.ActivityTypeProjectUpdate,
        activity.ResourceTypeProject,
        &project.ID,
        map[string]interface{}{
            "attempted_changes": changes,
            "error": err.Error(),
        },
        false,
        c.ClientIP(),
        c.Request.UserAgent(),
        err.Error(),
    )
    return
}

// Log successful update (only if there were changes)
if len(changes) > 0 {
    userID, username := h.activityService.ExtractUserInfoFromContext(c)
    h.activityService.LogCRUDActivity(
        c.Request.Context(),
        userID,
        username,
        activity.ActivityTypeProjectUpdate,
        activity.ResourceTypeProject,
        &project.ID,
        map[string]interface{}{
            "changes":    changes,
            "old_values": oldValues,
            "new_values": newValues,
        },
        true,
        c.ClientIP(),
        c.Request.UserAgent(),
        "",
    )
}
```

## Security and Privacy Considerations

1. **Sensitive Data**: The system automatically excludes sensitive data like passwords from logs
2. **IP Address Logging**: User IP addresses are logged for security auditing
3. **Data Retention**: Old activity logs can be cleaned up using `CleanupOldActivities()`
4. **Access Control**: Activity log endpoints require authentication
5. **Performance**: Asynchronous logging prevents impact on request performance

## Performance Optimizations

1. **Asynchronous Logging**: Activity logging doesn't block request processing
2. **Batch Operations**: Support for batch logging operations
3. **Database Indexing**: Optimized indexes on frequently queried fields
4. **Pagination**: All list operations support pagination to handle large datasets
5. **Selective Logging**: Configurable logging to skip health checks and static assets

## Configuration

### Middleware Options

```go
options := middleware.ActivityLoggerOptions{
    ShouldLog: func(c *gin.Context) bool {
        // Custom logic to determine if request should be logged
        return !strings.HasPrefix(c.Request.URL.Path, "/health")
    },
    IncludeBody: false,
    SkipPaths: []string{"/health", "/metrics"},
    SkipMethods: []string{"OPTIONS"},
    MaxBodySize: 10 * 1024, // 10KB
}
```

### Cleanup Configuration

```go
// Clean up activities older than 90 days
err := activityService.CleanupOldActivities(ctx, 90*24*time.Hour)
```

## Monitoring and Alerting

The activity logging system can be used for:

1. **Security Monitoring**: Failed login attempts, unusual access patterns
2. **Compliance Auditing**: Complete audit trails for regulatory requirements
3. **Performance Analysis**: API response times and error rates
4. **User Behavior Analysis**: Usage patterns and feature adoption
5. **System Health**: Error patterns and system performance metrics

## Best Practices

1. **Log Meaningful Activities**: Focus on actions that matter for auditing and security
2. **Include Context**: Always include relevant context information
3. **Handle Errors Gracefully**: Activity logging failures shouldn't break the main functionality
4. **Regular Cleanup**: Implement regular cleanup of old activity logs
5. **Monitor Performance**: Monitor the impact of activity logging on system performance
6. **Test Thoroughly**: Ensure activity logging works correctly in all scenarios

## Troubleshooting

### Common Issues

1. **High Database Load**: Check if too many activities are being logged
2. **Missing Activities**: Verify middleware is properly configured
3. **Performance Impact**: Consider asynchronous logging or sampling
4. **Storage Growth**: Implement regular cleanup of old activities

### Debug Information

- Check application logs for activity logging errors
- Verify database connectivity and permissions
- Monitor activity log table size and growth rate
- Test activity logging in development environment first

This comprehensive activity logging system provides complete visibility into user actions and system events, supporting security, compliance, and operational monitoring requirements.