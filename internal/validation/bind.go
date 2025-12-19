package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	validatorv10 "github.com/go-playground/validator/v10"
)

// BindAndValidate binds JSON body into `out` and runs validation.
// If validation fails, it writes a 400 response and returns an error for the handler to short-circuit.
func BindAndValidate(c *gin.Context, out interface{}, v *validatorv10.Validate) error {
	if err := c.ShouldBindJSON(out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid_request_body",
			"msg":   err.Error(),
		})
		return err
	}

	if err := v.Struct(out); err != nil {
		// return structured validation errors
		errs := validationErrorsToMap(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "validation_failed",
			"fields": errs,
		})
		return err
	}
	return nil
}

func validationErrorsToMap(err error) map[string]string {
	out := map[string]string{}
	if ve, ok := err.(validatorv10.ValidationErrors); ok {
		for _, fe := range ve {
			out[fe.StructNamespace()] = fe.Error() // simple message; can be improved
		}
	} else {
		out["error"] = err.Error()
	}
	return out
}
