# Create PRD and Detailed Task List

You are tasked with creating a comprehensive Product Requirements Document (PRD) and a detailed task list based on the conversation history with the user.

## Instructions

1. **Analyze Conversation History**
   - Review all messages exchanged with the user
   - Extract feature requirements, technical details, and implementation discussions
   - Identify the core problem being solved and the proposed solution

2. **Generate Short Description**
   - Create a concise slug (3-5 words, kebab-case) that describes the feature
   - Examples: `spx-tracking-integration`, `invoice-upload-feature`, `sicepat-awb-range`

3. **Create Directory Structure**
   - Create directory: `docs/{yyyy-mm-dd}-{short-desc}/`
   - Use today's date: 2026-02-25

4. **Generate PRD (`prd.md`)**
   - Follow the format in `docs/prd-format.md` EXACTLY
   - Fill in all relevant sections based on conversation:
     - Overview: Clear problem statement and solution
     - Goals and Non-Goals: What's in scope and out of scope
     - Functional Requirements: Detailed feature specifications
     - Technical Requirements: Architecture, database, repositories, usecases, controllers
     - Database Migrations: Specific SQL schemas discussed
     - Go Entities: Struct definitions
     - Validation Rules: Field validations
     - Error Handling: Error scenarios and messages
     - Testing Checklist: Comprehensive test cases
   - Include code snippets for schemas and entity definitions
   - Reference the Port-and-Adapter architecture pattern from CLAUDE.md

5. **Generate Detailed Task List (`task-progress.md`)**
   - Follow the format in `docs/task-progress-format.md` EXACTLY
   - Break down implementation into phases following this structure:

     **Phase 1: Database & Migrations**
     - [ ] Create migration file `migrations/YYYYMMDDHHMMSS_description.up.sql`
       - [ ] Write SQL schema for table X (include exact schema from PRD)
       - [ ] Write SQL schema for table Y (include exact schema from PRD)
       - [ ] Add indexes on columns: [specific columns]
       - [ ] Add foreign key constraints
     - [ ] Create migration file `migrations/YYYYMMDDHHMMSS_description.down.sql`
       - [ ] Write rollback SQL
     - [ ] Test migration: `make test-migration`
     - [ ] Run migration: `make run-migration`

     **Phase 2: Entities & Types**
     - [ ] Create `services/{domain}/type.go` (or update existing)
       - [ ] Define `type EntityName struct` with exact fields:
         ```go
         type EntityName struct {
             FieldName  Type  `db:"field_name"`
             // ... list all fields from PRD
         }
         ```
       - [ ] Add validation tags using `github.com/go-playground/validator/v10`
       - [ ] Add JSON tags if needed for API responses

     **Phase 3: Repository Layer**
     - [ ] Create repository interface in `services/{domain}/repo.go`
       - [ ] Define interface method: `MethodName(ctx context.Context, param ParamType) (ReturnType, error)`
       - [ ] Add method signature for each data operation
     - [ ] Create repository implementation in `services/{domain}/repo_{source}/repo.go`
       - [ ] Implement `MethodName` using sqlx and squirrel
       - [ ] Use `squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)` for PostgreSQL
       - [ ] Wrap errors with `fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)`
       - [ ] Use transactions where needed: `tx, err := r.db.BeginTxx(ctx, nil)`
       - [ ] Add proper connection context handling
     - [ ] Create `services/{domain}/repo_{source}/repo_test.go`
       - [ ] Set up test suite using `testify/suite`
       - [ ] Write test for each repository method
       - [ ] Use test fixtures from `test_data/`
       - [ ] Test error cases
     - [ ] Run repository tests: `go t[118;1:3uest -count=1 -failfast ./services/{domain}/repo_{source}/...`

     **Phase 4: Service Layer**
     - [ ] Update service interface in `services/{domain}/service.go`
       - [ ] Add method signature: `OperationName(ctx context.Context, param ParamType) (ReturnType, error)`
     - [ ] Implement service method in `services/{domain}/service.go`
       - [ ] Add business logic and validation
       - [ ] Call repository methods
       - [ ] Handle errors using `errors.Is()` for domain errors
       - [ ] Return appropriate domain-specific errors from `error.go`
     - [ ] Update `services/{domain}/error.go` if new errors needed
       - [ ] Define error variables: `var ErrSpecificCase = errors.New("error message")`
     - [ ] Update mocks: `mockery` (run from project root)
     - [ ] Create/update `services/{domain}/service_test.go`
       - [ ] Set up test suite with mocked repositories
       - [ ] Test happy path
       - [ ] Test error scenarios
       - [ ] Test edge cases
       - [ ] Verify repository method calls with mock assertions
     - [ ] Run service tests: `go test -count=1 -failfast ./services/{domain}/...`

     **Phase 5: HTTP Controller Layer**
     - [ ] Create `app/http/controller/{domain}/type.go` (if not exists)
       - [ ] Define request struct with validation tags
         ```go
         type OperationRequest struct {
             FieldName Type `json:"field_name" binding:"required"`
         }
         ```
       - [ ] Define response struct
     - [ ] Create/update `app/http/controller/{domain}/controller.go`
       - [ ] Add service dependency to controller struct
       - [ ] Implement handler method: `func (c *controller) HandlerName(ctx *gin.Context)`
       - [ ] Parse and validate request: `ctx.ShouldBindJSON(&req)`
       - [ ] Call service method
       - [ ] Handle errors with `c.handleError(ctx, err)`
       - [ ] Return success response using `github.com/useflik/shared/response`
     - [ ] Update `app/http/controller/{domain}/error.go`
       - [ ] Map service errors to HTTP status codes in `handleError()`

     **Phase 6: Views (if UI needed)**
     - [ ] Create `app/http/views/{domain}/{view_name}.templ`
       - [ ] Define templ component
       - [ ] Add HTML structure
       - [ ] Add Tailwind CSS classes
       - [ ] Add HTMX attributes if needed
       - [ ] Add form with proper validation
     - [ ] Generate Go code: `make generate` (runs templ generate and builds CSS)

     **Phase 7: Integration & Wiring**
     - [ ] Update `app/http/main.go`
       - [ ] Initialize repository: `repo := repo_postgres.New(db)`
       - [ ] Initialize service: `svc := service.New(repo)`
       - [ ] Initialize controller: `ctrl := controller.New(svc)`
       - [ ] Pass controller to router
     - [ ] Update `app/router.go`
       - [ ] Add route: `router.Method("/path", controller.HandlerName)`
       - [ ] Add middleware if needed
       - [ ] Group routes if appropriate

     **Phase 8: Testing & Validation**
     - [ ] Run all tests: `make test`
     - [ ] Run linter: `make run-lint`
     - [ ] Manual testing:
       - [ ] Test happy path with valid data
       - [ ] Test validation errors with invalid data
       - [ ] Test error scenarios (database errors, business logic errors)
       - [ ] Test edge cases
       - [ ] Test UI/UX flow (if applicable)
       - [ ] Test with different user roles (if applicable)
     - [ ] Verify database state after operations
     - [ ] Check logs for proper error messages

     **Phase 9: Documentation & Cleanup**
     - [ ] Update API documentation if needed
     - [ ] Add code comments for complex logic
     - [ ] Remove debug logging
     - [ ] Clean up unused imports
     - [ ] Verify all files are formatted: `gofmt -w .`

   - **Key Requirements for AI Agent Execution:**
     - Every task MUST include the exact file path
     - Every struct/function MUST include the signature
     - Every database operation MUST include the SQL or squirrel query pattern
     - Every test MUST specify what to test
     - Reference existing patterns: "Follow pattern in services/courier/repo_postgres/repo.go"
     - Include imports needed: "Use `github.com/Masterminds/squirrel`"
     - Include error handling pattern: "Wrap with util.GetPackageAndFunctionName()"
     - Specify test commands to run after each phase
     - Break complex tasks into sub-tasks with [ ] checkboxes

6. **Save Files**
   - Save PRD as: `docs/{yyyy-mm-dd}-{short-desc}/prd.md`
   - Save task list as: `docs/{yyyy-mm-dd}-{short-desc}/task-progress.md`

7. **Confirm Completion**
   - Show the user the directory path created
   - Show summary of PRD sections filled
   - Show total number of tasks created
   - Show estimated implementation phases

## Important Notes

- **Be Exhaustive**: Don't skip details. AI agents need explicit instructions.
- **Use Exact Paths**: Always include full file paths relative to project root.
- **Reference Architecture**: Follow the Port-and-Adapter pattern from CLAUDE.md.
- **Follow Patterns**: Reference existing implementations in the codebase.
- **Include Test Commands**: Specify exact commands to run after each phase.
- **Break Down Complex Tasks**: If a task is complex, break it into 5-10 sub-tasks.
- **Add Context**: If a task requires understanding existing code, note which files to read first.

## Example Task Breakdown

Good (AI-executable):
```markdown
- [ ] Create repository interface in `services/shipment/repo.go`
  - [ ] Add method signature: `GetShipmentByID(ctx context.Context, id int64) (*Shipment, error)`
  - [ ] Add method signature: `CreateShipment(ctx context.Context, shipment *Shipment) error`
  - [ ] Add method signature: `UpdateShipmentStatus(ctx context.Context, id int64, status string) error`
```

Bad (too vague):
```markdown
- [ ] Create repository
- [ ] Add methods
```

Now proceed with creating the PRD and detailed task list based on our conversation history.

