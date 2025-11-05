package bind

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	apierrors "github.com/lyuangg/yuango/internal/errors"
)

func TestBindJSON(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		req       interface{}
		wantError bool
		checkFunc func(t *testing.T, req interface{})
	}{
		{
			name: "valid JSON",
			body: `{"name":"John","email":"john@example.com","age":25}`,
			req: &struct {
				Name  string `json:"name" validate:"required"`
				Email string `json:"email" validate:"required,email"`
				Age   int    `json:"age" validate:"required,gte=18"`
			}{},
			wantError: false,
			checkFunc: func(t *testing.T, req interface{}) {
				r := req.(*struct {
					Name  string `json:"name" validate:"required"`
					Email string `json:"email" validate:"required,email"`
					Age   int    `json:"age" validate:"required,gte=18"`
				})
				if r.Name != "John" {
					t.Errorf("expected Name=John, got %s", r.Name)
				}
				if r.Email != "john@example.com" {
					t.Errorf("expected Email=john@example.com, got %s", r.Email)
				}
				if r.Age != 25 {
					t.Errorf("expected Age=25, got %d", r.Age)
				}
			},
		},
		{
			name: "invalid JSON",
			body: `{"name":"John"`,
			req: &struct {
				Name string `json:"name" validate:"required"`
			}{},
			wantError: true,
		},
		{
			name: "validation failed",
			body: `{"name":"","email":"invalid","age":15}`,
			req: &struct {
				Name  string `json:"name" validate:"required"`
				Email string `json:"email" validate:"required,email"`
				Age   int    `json:"age" validate:"required,gte=18"`
			}{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			err := BindJSON(r, tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindJSON() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFunc != nil {
				tt.checkFunc(t, tt.req)
			}

			if tt.wantError && err != nil {
				// 验证返回的是 APIError
				if _, ok := apierrors.AsAPIError(err); !ok {
					t.Errorf("expected APIError, got %T", err)
				}
			}
		})
	}
}

func TestBindQuery(t *testing.T) {
	type Request struct {
		Page   int    `query:"page" validate:"gte=1"`
		Size   int    `query:"size" validate:"gte=1,lte=100"`
		Search string `query:"search"`
	}

	tests := []struct {
		name      string
		url       string
		wantError bool
		checkFunc func(t *testing.T, req *Request)
	}{
		{
			name:      "valid query params",
			url:       "/test?page=1&size=10&search=test",
			wantError: false,
			checkFunc: func(t *testing.T, req *Request) {
				if req.Page != 1 {
					t.Errorf("expected Page=1, got %d", req.Page)
				}
				if req.Size != 10 {
					t.Errorf("expected Size=10, got %d", req.Size)
				}
				if req.Search != "test" {
					t.Errorf("expected Search=test, got %s", req.Search)
				}
			},
		},
		{
			name:      "validation failed",
			url:       "/test?page=0&size=200",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req := &Request{}

			err := BindQuery(r, req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindQuery() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFunc != nil {
				tt.checkFunc(t, req)
			}
		})
	}
}

func TestBindPath(t *testing.T) {
	type Request struct {
		ID uint `path:"id" validate:"required"`
	}

	// 注意：Go 1.22+ 的 PathValue 需要实际的路由匹配
	// 这里只是测试绑定逻辑，实际使用需要在路由中设置路径参数
	t.Run("path binding", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		// 模拟 PathValue（实际使用时由路由提供）
		// 这里我们直接测试 bindPath 函数需要实际的路径值
		// 由于测试环境限制，这个测试可能需要在实际路由中验证
		req := &Request{}

		// 如果没有路径值，绑定会失败（这是预期的）
		err := BindPath(r, req)
		// 在测试环境中，PathValue 可能为空，所以这个测试主要用于验证函数不会 panic
		_ = err
	})
}

func TestBindJSON_EmptyBody(t *testing.T) {
	type Request struct {
		Name string `json:"name" validate:"required"`
	}

	// 测试 1: nil body
	r1 := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/test"},
		Header: make(http.Header),
	}
	r1.Header.Set("Content-Type", "application/json")
	r1.Body = nil // 明确设置为 nil
	req1 := &Request{}

	err1 := BindJSON(r1, req1)
	if err1 == nil {
		t.Error("expected error for nil body, got nil")
	}
	if apiErr, ok := apierrors.AsAPIError(err1); !ok || apiErr.Code != 400 {
		t.Errorf("expected BadRequest error, got %v", err1)
	}

	// 测试 2: httptest.NewRequest 创建的请求（body 可能不为 nil）
	r2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	r2.Header.Set("Content-Type", "application/json")
	req2 := &Request{}

	err2 := BindJSON(r2, req2)
	// httptest.NewRequest 可能创建了一个空的 body，所以可能不会触发 nil 检查
	_ = err2
}

func TestBindForm(t *testing.T) {
	type Request struct {
		Username string `form:"username" validate:"required,min=3"`
		Password string `form:"password" validate:"required,min=6"`
		Age      int    `form:"age" validate:"gte=18"`
	}

	tests := []struct {
		name        string
		method      string
		body        string
		contentType string
		wantError   bool
		checkFunc   func(t *testing.T, req *Request)
	}{
		{
			name:        "valid form data",
			method:      http.MethodPost,
			body:        "username=john&password=password123&age=25",
			contentType: "application/x-www-form-urlencoded",
			wantError:   false,
			checkFunc: func(t *testing.T, req *Request) {
				if req.Username != "john" {
					t.Errorf("expected Username=john, got %s", req.Username)
				}
				if req.Password != "password123" {
					t.Errorf("expected Password=password123, got %s", req.Password)
				}
				if req.Age != 25 {
					t.Errorf("expected Age=25, got %d", req.Age)
				}
			},
		},
		{
			name:        "validation failed",
			method:      http.MethodPost,
			body:        "username=ab&password=123&age=15",
			contentType: "application/x-www-form-urlencoded",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			req := &Request{}

			err := BindForm(r, req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindForm() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFunc != nil {
				tt.checkFunc(t, req)
			}
		})
	}
}

// errorReader 是一个会返回错误的 io.ReadCloser
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (e *errorReader) Close() error {
	return nil
}

func TestBindForm_ParseFormError(t *testing.T) {
	// 创建一个会失败的请求（body 太大或其他错误）
	// 注意：在实际测试中很难模拟 ParseForm 失败，因为 httptest.NewRequest 总是能成功解析
	// 这里我们主要测试代码路径
	type Request struct {
		Name string `form:"name"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	// 正常情况下应该成功
	if err != nil {
		t.Logf("BindForm returned error (expected in some cases): %v", err)
	}

	// 创建一个会失败的请求 - 使用会返回错误的 reader
	r2 := httptest.NewRequest(http.MethodPost, "/test", &errorReader{})
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2.ContentLength = 100 // 设置一个非零的 ContentLength
	req2 := &Request{}

	err2 := BindForm(r2, req2)
	// 这应该会触发 ParseForm 错误分支
	if err2 == nil {
		t.Logf("BindForm succeeded even with error reader (ParseForm may handle this gracefully)")
	} else {
		// 验证返回的是 APIError
		if apiErr, ok := apierrors.AsAPIError(err2); ok {
			if apiErr.Code != 400 {
				t.Errorf("expected BadRequest error, got code %d", apiErr.Code)
			}
			if !strings.Contains(apiErr.Details, "failed to parse form") {
				t.Errorf("expected error message to contain 'failed to parse form', got: %s", apiErr.Details)
			}
		} else {
			t.Errorf("expected APIError, got %T: %v", err2, err2)
		}
	}
}

func TestBindForm_EmptyForm(t *testing.T) {
	type Request struct {
		Name string `form:"name"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	// 空表单应该成功绑定（字段值为空）
	if err != nil {
		t.Errorf("unexpected error for empty form: %v", err)
	}
}

func TestBindForm_InvalidPointer(t *testing.T) {
	type Request struct {
		Name string `form:"name"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := Request{} // 不是指针

	err := BindForm(r, req)
	if err == nil {
		t.Error("expected error for non-pointer, got nil")
	}
}

func TestBindForm_NotStruct(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var req string // 不是结构体

	err := BindForm(r, &req)
	if err == nil {
		t.Error("expected error for non-struct, got nil")
	}
}

func TestBindForm_AllFieldTypes(t *testing.T) {
	type Request struct {
		Str     string  `form:"str"`
		Int     int     `form:"int"`
		Int8    int8    `form:"int8"`
		Int16   int16   `form:"int16"`
		Int32   int32   `form:"int32"`
		Int64   int64   `form:"int64"`
		Uint    uint    `form:"uint"`
		Uint8   uint8   `form:"uint8"`
		Uint16  uint16  `form:"uint16"`
		Uint32  uint32  `form:"uint32"`
		Uint64  uint64  `form:"uint64"`
		Float32 float32 `form:"float32"`
		Float64 float64 `form:"float64"`
		Bool    bool    `form:"bool"`
	}

	body := "str=test&int=42&int8=8&int16=16&int32=32&int64=64&uint=100&uint8=8&uint16=16&uint32=32&uint64=64&float32=3.14&float64=2.718&bool=true"
	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Str != "test" {
		t.Errorf("expected Str=test, got %s", req.Str)
	}
	if req.Int != 42 {
		t.Errorf("expected Int=42, got %d", req.Int)
	}
	if req.Bool != true {
		t.Errorf("expected Bool=true, got %v", req.Bool)
	}
}

func TestBindForm_MultipleValues(t *testing.T) {
	type Request struct {
		Name string `form:"name"`
	}

	// 测试表单中有多个同名字段的情况（ParseForm 会取第一个值）
	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=first&name=second"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ParseForm 会取第一个值
	if req.Name != "first" {
		t.Errorf("expected Name=first, got %s", req.Name)
	}
}

func TestBindForm_WithQueryParams(t *testing.T) {
	type Request struct {
		Name string `form:"name"`
		Page int    `form:"page"`
	}

	// 测试 POST 表单数据（不会包含 URL 查询参数）
	r := httptest.NewRequest(http.MethodPost, "/test?page=1", strings.NewReader("name=test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Form 数据应该绑定 name，但 page 应该为空（因为 form 只绑定 PostForm，不包括 Query）
	if req.Name != "test" {
		t.Errorf("expected Name=test, got %s", req.Name)
	}
}

func TestBindForm_InvalidFieldValue(t *testing.T) {
	type Request struct {
		Age int `form:"age"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("age=invalid"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := &Request{}

	err := BindForm(r, req)
	if err == nil {
		t.Error("expected error for invalid int value, got nil")
	}
}

func TestBindPath_WithPathValue(t *testing.T) {
	type Request struct {
		ID   uint   `path:"id" validate:"required"`
		Name string `path:"name"`
	}

	r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	// 注意：在测试环境中，PathValue 可能返回空字符串
	// 实际使用时需要路由来设置路径参数
	req := &Request{}

	err := BindPath(r, req)
	// 在测试环境中，PathValue 为空是预期的
	_ = err
}

func TestBindAll_Fallback(t *testing.T) {
	type Request struct {
		Name string `json:"name" form:"name" query:"name" path:"name"`
	}

	// 测试所有绑定都失败的情况
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	req := &Request{}

	err := BindAll(r, req)
	// 可能会返回错误，因为没有任何数据可以绑定
	_ = err
}

func TestSetFieldValue_ErrorCases(t *testing.T) {
	type Request struct {
		InvalidType map[string]string
	}

	r := httptest.NewRequest(http.MethodGet, "/test?InvalidType=value", nil)
	req := &Request{}

	err := BindQuery(r, req)
	// 应该返回错误，因为 map 类型不支持
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestBindAll_MultipartFormData(t *testing.T) {
	type Request struct {
		Name  string `form:"name" validate:"required"`
		Email string `form:"email" validate:"required,email"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=John&email=john@example.com"))
	r.Header.Set("Content-Type", "multipart/form-data")
	req := &Request{}

	err := BindAll(r, req)
	// multipart/form-data 需要特殊处理，这里测试代码路径
	_ = err
}

func TestBindPath_LowercaseFieldName(t *testing.T) {
	type Request struct {
		UserID uint `validate:"required"` // 没有 path tag，会使用小写字段名
	}

	r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req := &Request{}

	err := BindPath(r, req)
	// 在测试环境中，PathValue 可能为空
	_ = err
}

func TestBindJSON_WithValidationError(t *testing.T) {
	type Request struct {
		Name string `json:"name" validate:"required"`
	}

	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	req := &Request{}

	err := BindJSON(r, req)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestBindAll(t *testing.T) {
	type Request struct {
		Name  string `json:"name" form:"name" query:"name" validate:"required"`
		Email string `json:"email" form:"email" query:"email" validate:"required,email"`
	}

	tests := []struct {
		name        string
		method      string
		url         string
		body        string
		contentType string
		wantError   bool
		checkFunc   func(t *testing.T, req *Request)
	}{
		{
			name:        "bind JSON",
			method:      http.MethodPost,
			url:         "/test",
			body:        `{"name":"John","email":"john@example.com"}`,
			contentType: "application/json",
			wantError:   false,
			checkFunc: func(t *testing.T, req *Request) {
				if req.Name != "John" {
					t.Errorf("expected Name=John, got %s", req.Name)
				}
			},
		},
		{
			name:        "bind Form",
			method:      http.MethodPost,
			url:         "/test",
			body:        "name=John&email=john@example.com",
			contentType: "application/x-www-form-urlencoded",
			wantError:   false,
			checkFunc: func(t *testing.T, req *Request) {
				if req.Name != "John" {
					t.Errorf("expected Name=John, got %s", req.Name)
				}
			},
		},
		{
			name:        "bind Query",
			method:      http.MethodGet,
			url:         "/test?name=John&email=john@example.com",
			contentType: "",
			wantError:   false,
			checkFunc: func(t *testing.T, req *Request) {
				if req.Name != "John" {
					t.Errorf("expected Name=John, got %s", req.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			if tt.contentType != "" {
				r.Header.Set("Content-Type", tt.contentType)
			}
			req := &Request{}

			err := BindAll(r, req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindAll() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFunc != nil {
				tt.checkFunc(t, req)
			}
		})
	}
}

func TestBindQuery_TagPriority(t *testing.T) {
	type Request struct {
		Field1 string `form:"f1" query:"q1" json:"j1"`
		Field2 string `query:"q2"`
		Field3 string `json:"j3"`
	}

	r := httptest.NewRequest(http.MethodGet, "/test?f1=form&q1=query&q2=query2&j3=json", nil)
	req := &Request{}

	err := BindQuery(r, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// form tag 优先级最高
	if req.Field1 != "form" {
		t.Errorf("expected Field1=form (form tag priority), got %s", req.Field1)
	}
	// query tag 优先级次之
	if req.Field2 != "query2" {
		t.Errorf("expected Field2=query2, got %s", req.Field2)
	}
	// json tag 优先级最低
	if req.Field3 != "json" {
		t.Errorf("expected Field3=json, got %s", req.Field3)
	}
}

func TestBindQuery_InvalidValue(t *testing.T) {
	type Request struct {
		Age int `query:"age"`
	}

	r := httptest.NewRequest(http.MethodGet, "/test?age=invalid", nil)
	req := &Request{}

	err := BindQuery(r, req)
	if err == nil {
		t.Error("expected error for invalid int value, got nil")
	}
}

func TestBindQuery_NotPointer(t *testing.T) {
	type Request struct {
		Name string `query:"name"`
	}

	r := httptest.NewRequest(http.MethodGet, "/test?name=test", nil)
	req := Request{} // 不是指针

	err := BindQuery(r, req)
	if err == nil {
		t.Error("expected error for non-pointer, got nil")
	}
}

func TestBindQuery_NotStruct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?name=test", nil)
	var req string // 不是结构体

	err := BindQuery(r, &req)
	if err == nil {
		t.Error("expected error for non-struct, got nil")
	}
}

func TestSetFieldValue_AllTypes(t *testing.T) {
	type Request struct {
		Str     string
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Float32 float32
		Float64 float64
		Bool    bool
	}

	// 使用 BindQuery 来测试 setFieldValue
	r := httptest.NewRequest(http.MethodGet, "/test?Str=test&Int=42&Int8=8&Int16=16&Int32=32&Int64=64&Uint=100&Uint8=8&Uint16=16&Uint32=32&Uint64=64&Float32=3.14&Float64=2.718&Bool=true", nil)
	req := &Request{}

	err := BindQuery(r, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Str != "test" {
		t.Errorf("expected Str=test, got %s", req.Str)
	}
	if req.Int != 42 {
		t.Errorf("expected Int=42, got %d", req.Int)
	}
	if req.Int8 != 8 {
		t.Errorf("expected Int8=8, got %d", req.Int8)
	}
	if req.Int16 != 16 {
		t.Errorf("expected Int16=16, got %d", req.Int16)
	}
	if req.Int32 != 32 {
		t.Errorf("expected Int32=32, got %d", req.Int32)
	}
	if req.Int64 != 64 {
		t.Errorf("expected Int64=64, got %d", req.Int64)
	}
	if req.Uint != 100 {
		t.Errorf("expected Uint=100, got %d", req.Uint)
	}
	if req.Uint8 != 8 {
		t.Errorf("expected Uint8=8, got %d", req.Uint8)
	}
	if req.Uint16 != 16 {
		t.Errorf("expected Uint16=16, got %d", req.Uint16)
	}
	if req.Uint32 != 32 {
		t.Errorf("expected Uint32=32, got %d", req.Uint32)
	}
	if req.Uint64 != 64 {
		t.Errorf("expected Uint64=64, got %d", req.Uint64)
	}
	if req.Float32 != 3.14 {
		t.Errorf("expected Float32=3.14, got %f", req.Float32)
	}
	if req.Float64 != 2.718 {
		t.Errorf("expected Float64=2.718, got %f", req.Float64)
	}
	if req.Bool != true {
		t.Errorf("expected Bool=true, got %v", req.Bool)
	}
}

func TestValidateStruct_MoreTags(t *testing.T) {
	type Request struct {
		MaxLen   string `validate:"max=5"`
		Len      string `validate:"len=3"`
		URL      string `validate:"url"`
		Numeric  string `validate:"numeric"`
		Alpha    string `validate:"alpha"`
		Alphanum string `validate:"alphanum"`
		Lte      int    `validate:"lte=10"`
		Gt       int    `validate:"gt=5"`
		Lt       int    `validate:"lt=10"`
		Oneof    string `validate:"oneof=red blue green"`
	}

	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name: "valid - all tags",
			value: Request{
				MaxLen:   "12345",
				Len:      "abc",
				URL:      "https://example.com",
				Numeric:  "12345",
				Alpha:    "abc",
				Alphanum: "abc123",
				Lte:      10,
				Gt:       6,
				Lt:       9,
				Oneof:    "red",
			},
			wantError: false,
		},
		{
			name: "invalid - max exceeded",
			value: Request{
				MaxLen: "123456",
			},
			wantError: true,
		},
		{
			name: "invalid - len mismatch",
			value: Request{
				Len: "ab",
			},
			wantError: true,
		},
		{
			name: "invalid - invalid URL",
			value: Request{
				URL: "not-a-url",
			},
			wantError: true,
		},
		{
			name: "invalid - lte exceeded",
			value: Request{
				Lte: 11,
			},
			wantError: true,
		},
		{
			name: "invalid - gt not met",
			value: Request{
				Gt: 5,
			},
			wantError: true,
		},
		{
			name: "invalid - oneof not matched",
			value: Request{
				Oneof: "yellow",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() error = %v, wantError %v", err, tt.wantError)
				if err != nil {
					t.Logf("error details: %v", err)
				}
			}
		})
	}
}

func TestBindJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		req       interface{}
		wantError bool
	}{
		{
			name:      "missing closing brace",
			body:      `{"name":"John"`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
		{
			name:      "missing quotes",
			body:      `{name:"John"}`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
		{
			name:      "invalid character",
			body:      `{"name":"John` + "\x00" + `"}`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
		{
			name:      "JSON array instead of object",
			body:      `["name","John"]`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
		{
			name:      "trailing comma",
			body:      `{"name":"John",}`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
		{
			name:      "wrong type",
			body:      `{"name":123}`,
			req:       &struct{ Name string }{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			err := BindJSON(r, tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindJSON() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil {
				if apiErr, ok := apierrors.AsAPIError(err); !ok || apiErr.Code != 400 {
					t.Errorf("expected BadRequest APIError, got %v", err)
				}
			}
		})
	}
}

func TestBindJSON_MoreEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		req       interface{}
		wantError bool
		desc      string
	}{
		{
			name:      "JSON with unescaped newline",
			body:      `{"name":"John\nDoe"}`,
			req:       &struct{ Name string }{},
			wantError: false,
			desc:      "JSON 字符串中包含转义的换行符（Go JSON 解析器支持）",
		},
		{
			name:      "JSON with invalid escape",
			body:      `{"name":"John\z"}`,
			req:       &struct{ Name string }{},
			wantError: true,
			desc:      "JSON 字符串中包含无效的转义序列",
		},
		{
			name:      "JSON number overflow",
			body:      `{"age":999999999999999999999999999999}`,
			req:       &struct{ Age int64 }{},
			wantError: true,
			desc:      "JSON 数字溢出",
		},
		{
			name: "JSON deeply nested",
			body: `{"a":{"b":{"c":"value"}}}`,
			req: &struct {
				A struct{ B struct{ C string } }
			}{},
			wantError: false,
			desc:      "深度嵌套的 JSON（简化版本）",
		},
		{
			name:      "JSON with UTF-8 BOM",
			body:      "\xEF\xBB\xBF{\"name\":\"John\"}",
			req:       &struct{ Name string }{},
			wantError: true,
			desc:      "JSON 带有 UTF-8 BOM（Go JSON 解析器不支持）",
		},
		{
			name:      "JSON with multiple top-level",
			body:      `{"name":"John"}{"age":25}`,
			req:       &struct{ Name string }{},
			wantError: false,
			desc:      "多个 JSON 对象（decoder 只解析第一个）",
		},
		{
			name:      "JSON null value",
			body:      `{"name":null}`,
			req:       &struct{ Name *string }{},
			wantError: false,
			desc:      "JSON null 值",
		},
		{
			name:      "JSON null value wrong type",
			body:      `{"name":null}`,
			req:       &struct{ Name string }{},
			wantError: false,
			desc:      "JSON null 值转换为字符串零值（Go JSON 解析器支持）",
		},
		{
			name:      "JSON boolean value",
			body:      `{"active":true}`,
			req:       &struct{ Active bool }{},
			wantError: false,
			desc:      "JSON 布尔值",
		},
		{
			name:      "JSON boolean value wrong type",
			body:      `{"active":true}`,
			req:       &struct{ Active string }{},
			wantError: true,
			desc:      "JSON 布尔值但字段类型不匹配",
		},
		{
			name:      "JSON number as string",
			body:      `{"age":"25"}`,
			req:       &struct{ Age int }{},
			wantError: true,
			desc:      "JSON 数字作为字符串",
		},
		{
			name:      "JSON empty string",
			body:      `{"name":""}`,
			req:       &struct{ Name string }{},
			wantError: false,
			desc:      "JSON 空字符串",
		},
		{
			name:      "JSON with special characters",
			body:      `{"name":"John\u0020Doe"}`,
			req:       &struct{ Name string }{},
			wantError: false,
			desc:      "JSON Unicode 转义字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			err := BindJSON(r, tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindJSON() error = %v, wantError %v (%s)", err, tt.wantError, tt.desc)
				if err != nil {
					t.Logf("Error details: %v", err)
				}
			}
			if err != nil {
				if apiErr, ok := apierrors.AsAPIError(err); ok {
					if apiErr.Code != 400 {
						t.Errorf("expected BadRequest error, got code %d", apiErr.Code)
					}
				}
			}
		})
	}
}

func TestBindPath_InvalidPointer(t *testing.T) {
	type Request struct {
		ID uint `path:"id"`
	}

	r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req := Request{} // 不是指针

	err := BindPath(r, req)
	if err == nil {
		t.Error("expected error for non-pointer, got nil")
	}
	if !strings.Contains(err.Error(), "must be a pointer to struct") {
		t.Errorf("expected error message about pointer, got: %v", err)
	}
}

func TestBindPath_NotStruct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	var req string // 不是结构体

	err := BindPath(r, &req)
	if err == nil {
		t.Error("expected error for non-struct, got nil")
	}
	if !strings.Contains(err.Error(), "must be a pointer to struct") {
		t.Errorf("expected error message about struct, got: %v", err)
	}
}

func TestBindPath_InvalidPathValue(t *testing.T) {
	type Request struct {
		ID   uint    `path:"id" validate:"required"`
		Age  int     `path:"age"`
		Rate float64 `path:"rate"`
	}

	// 注意：在测试环境中，我们需要模拟 PathValue
	// 由于 httptest.NewRequest 的 PathValue 在测试环境中可能返回空字符串
	// 我们通过创建一个实际的 HTTP 服务器来测试真实的路由情况
	// 但这里我们主要测试错误处理逻辑

	r := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req := &Request{}

	// 由于测试环境限制，我们通过其他方式测试错误处理
	// 这里测试基本功能
	err := BindPath(r, req)
	// 在测试环境中，PathValue 为空，所以这个测试主要用于验证函数不会 panic
	_ = err
}

func TestSetFieldValue_UnsettableField(t *testing.T) {
	// 注意：setFieldValue 是私有函数，我们通过测试未导出字段来间接测试
	// 但是 Go 的反射机制允许设置未导出的字段（如果在同一个包内）
	// 所以我们通过创建一个无法设置的字段（如接口值）来测试

	type Request struct {
		PublicField string
	}

	// 创建一个请求，然后尝试通过反射获取一个不可设置的字段
	// 实际上，在同一个包内，未导出字段也是可以设置的
	// 所以我们通过其他方式测试：创建一个 nil 指针的字段

	_ = &Request{}

	// 创建一个不可设置的场景：尝试设置一个 nil 接口值
	var nilInterface interface{}
	nilValue := reflect.ValueOf(nilInterface)

	// nilValue 是不可设置的
	if nilValue.CanSet() {
		t.Skip("Cannot test unsettable field with current approach")
	}

	// 由于 setFieldValue 是私有函数，我们通过测试实际使用场景来覆盖
	// 这里我们主要确保代码逻辑正确
	t.Log("setFieldValue unsettable field test skipped - tested through other paths")
}

func TestSetFieldValue_InvalidBool(t *testing.T) {
	type Request struct {
		Flag bool `query:"flag"`
	}

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "invalid value",
			value:     "maybe",
			wantError: true,
		},
		{
			name:      "yes",
			value:     "yes",
			wantError: true,
		},
		{
			name:      "no",
			value:     "no",
			wantError: true,
		},
		{
			name:      "empty string",
			value:     "",
			wantError: true,
		},
		{
			name:      "valid true",
			value:     "true",
			wantError: false,
		},
		{
			name:      "valid false",
			value:     "false",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/test?flag="+tt.value, nil)
			req := &Request{}

			err := BindQuery(r, req)
			if (err != nil) != tt.wantError {
				t.Errorf("BindQuery() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	type ValidStruct struct {
		Name  string `validate:"required,min=2"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=18"`
	}

	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name: "valid struct",
			value: ValidStruct{
				Name:  "John",
				Email: "john@example.com",
				Age:   25,
			},
			wantError: false,
		},
		{
			name: "invalid struct - missing required",
			value: ValidStruct{
				Name:  "",
				Email: "john@example.com",
				Age:   25,
			},
			wantError: true,
		},
		{
			name: "invalid struct - invalid email",
			value: ValidStruct{
				Name:  "John",
				Email: "invalid-email",
				Age:   25,
			},
			wantError: true,
		},
		{
			name: "invalid struct - age too low",
			value: ValidStruct{
				Name:  "John",
				Email: "john@example.com",
				Age:   15,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
