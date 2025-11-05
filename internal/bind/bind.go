// Package bind provides utilities for binding HTTP request data to structs with validation.
package bind

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	apierrors "github.com/lyuangg/yuango/internal/errors"
)

// validatorInstance 是全局验证器实例
var validatorInstance *validator.Validate

func init() {
	validatorInstance = validator.New()
	// 注册自定义验证器可以在这里添加
}

// BindJSON 从请求体中绑定 JSON 数据到结构体并验证
func BindJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return apierrors.NewWithDetails(
			http.StatusBadRequest,
			"Bad Request",
			"request body is empty",
		)
	}

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return apierrors.NewWithDetails(
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("invalid JSON format: %v", err),
		)
	}

	return ValidateStruct(dst)
}

// BindQuery 从 URL 查询参数绑定到结构体并验证
func BindQuery(r *http.Request, dst interface{}) error {
	return bindQuery(r.URL.Query(), dst)
}

// BindForm 从表单数据绑定到结构体并验证
func BindForm(r *http.Request, dst interface{}) error {
	if err := r.ParseForm(); err != nil {
		return apierrors.NewWithDetails(
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("failed to parse form: %v", err),
		)
	}
	return bindQuery(r.PostForm, dst)
}

// BindPath 从 URL 路径参数绑定到结构体并验证
// 适用于 Go 1.22+ 的 PathValue
func BindPath(r *http.Request, dst interface{}) error {
	return bindPath(r, dst)
}

// BindAll 尝试绑定所有可能的来源（JSON body、Query、Form、Path）
// 优先级：JSON > Form > Query > Path
func BindAll(r *http.Request, dst interface{}) error {
	contentType := r.Header.Get("Content-Type")

	// 1. 尝试绑定 JSON（如果 Content-Type 是 application/json）
	if strings.HasPrefix(contentType, "application/json") {
		if err := BindJSON(r, dst); err == nil {
			return nil
		}
	}

	// 2. 尝试绑定 Form（如果 Content-Type 是 application/x-www-form-urlencoded 或 multipart/form-data）
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(contentType, "multipart/form-data") {
		if err := BindForm(r, dst); err == nil {
			return nil
		}
	}

	// 3. 尝试绑定 Query
	if err := BindQuery(r, dst); err == nil {
		return nil
	}

	// 4. 尝试绑定 Path
	if err := BindPath(r, dst); err == nil {
		return nil
	}

	return apierrors.NewWithDetails(
		http.StatusBadRequest,
		"Bad Request",
		"failed to bind request data",
	)
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	if err := validatorInstance.Struct(s); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		return formatValidationErrors(validationErrors)
	}
	return nil
}

// formatValidationErrors 格式化验证错误为友好的错误信息
func formatValidationErrors(errs validator.ValidationErrors) *apierrors.APIError {
	var messages []string
	for _, err := range errs {
		field := err.Field()
		tag := err.Tag()
		param := err.Param()

		var msg string
		switch tag {
		case "required":
			msg = fmt.Sprintf("%s is required", field)
		case "min":
			msg = fmt.Sprintf("%s must be at least %s", field, param)
		case "max":
			msg = fmt.Sprintf("%s must be at most %s", field, param)
		case "len":
			msg = fmt.Sprintf("%s must be exactly %s characters", field, param)
		case "email":
			msg = fmt.Sprintf("%s must be a valid email address", field)
		case "url":
			msg = fmt.Sprintf("%s must be a valid URL", field)
		case "numeric":
			msg = fmt.Sprintf("%s must be a number", field)
		case "alpha":
			msg = fmt.Sprintf("%s must contain only letters", field)
		case "alphanum":
			msg = fmt.Sprintf("%s must contain only letters and numbers", field)
		case "gte":
			msg = fmt.Sprintf("%s must be greater than or equal to %s", field, param)
		case "lte":
			msg = fmt.Sprintf("%s must be less than or equal to %s", field, param)
		case "gt":
			msg = fmt.Sprintf("%s must be greater than %s", field, param)
		case "lt":
			msg = fmt.Sprintf("%s must be less than %s", field, param)
		case "oneof":
			msg = fmt.Sprintf("%s must be one of: %s", field, param)
		default:
			msg = fmt.Sprintf("%s failed validation for tag '%s'", field, tag)
		}

		messages = append(messages, msg)
	}

	return apierrors.NewWithDetails(
		http.StatusBadRequest,
		"Validation Failed",
		strings.Join(messages, "; "),
	)
}

// bindQuery 从 url.Values 绑定到结构体
func bindQuery(values map[string][]string, dst interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct")
	}

	dstValue = dstValue.Elem()
	dstType := dstValue.Type()

	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		fieldValue := dstValue.Field(i)

		// 获取字段标签
		formTag := field.Tag.Get("form")
		queryTag := field.Tag.Get("query")
		jsonTag := field.Tag.Get("json")

		// 确定字段名（优先级：form > query > json > 字段名）
		fieldName := field.Name
		if formTag != "" && formTag != "-" {
			fieldName = formTag
		} else if queryTag != "" && queryTag != "-" {
			fieldName = queryTag
		} else if jsonTag != "" && jsonTag != "-" {
			// 提取 json tag 名称（去掉 omitempty 等）
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// 尝试从 values 中获取值
		vals, exists := values[fieldName]
		if !exists || len(vals) == 0 {
			continue
		}

		// 设置字段值
		if err := setFieldValue(fieldValue, vals[0]); err != nil {
			return apierrors.NewWithDetails(
				http.StatusBadRequest,
				"Bad Request",
				fmt.Sprintf("invalid value for field '%s': %v", fieldName, err),
			)
		}
	}

	return ValidateStruct(dst)
}

// bindPath 从路径参数绑定到结构体
func bindPath(r *http.Request, dst interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct")
	}

	dstValue = dstValue.Elem()
	dstType := dstValue.Type()

	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		fieldValue := dstValue.Field(i)

		// 获取字段标签
		pathTag := field.Tag.Get("path")
		if pathTag == "" || pathTag == "-" {
			// 如果没有 path tag，尝试使用字段名的小写形式
			pathTag = strings.ToLower(field.Name)
		}

		// 从路径参数获取值
		pathValue := r.PathValue(pathTag)
		if pathValue == "" {
			continue
		}

		// 设置字段值
		if err := setFieldValue(fieldValue, pathValue); err != nil {
			return apierrors.NewWithDetails(
				http.StatusBadRequest,
				"Bad Request",
				fmt.Sprintf("invalid value for path parameter '%s': %v", pathTag, err),
			)
		}
	}

	return ValidateStruct(dst)
}

// setFieldValue 设置字段值
func setFieldValue(fieldValue reflect.Value, value string) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		fieldValue.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		fieldValue.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		fieldValue.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		fieldValue.SetBool(boolVal)

	default:
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}

	return nil
}
