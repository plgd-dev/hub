package errors_test

import (
	"errors"
	"fmt"
	"testing"

	pkgErrors "github.com/plgd-dev/hub/v2/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestWrappedError(t *testing.T) {
	testErr1 := fmt.Errorf("test1")

	err1 := pkgErrors.NewError("", testErr1)
	require.Equal(t, testErr1.Error(), err1.Error())
	require.True(t, errors.Is(err1, testErr1))

	err2 := pkgErrors.NewError("err2", testErr1)
	require.Equal(t, testErr1.Error()+": err2", err2.Error())
	require.True(t, errors.Is(err2, testErr1))

	testErr2 := fmt.Errorf("test2")
	require.False(t, errors.Is(err1, testErr2))
	require.False(t, errors.Is(err2, testErr2))

	err3 := pkgErrors.NewError("err3", testErr1, testErr2)
	require.Equal(t, testErr1.Error()+": err3: "+testErr2.Error(), err3.Error())
	require.True(t, errors.Is(err3, testErr1))
	require.True(t, errors.Is(err3, testErr2))
	require.False(t, errors.Is(err3, err1))
	require.False(t, errors.Is(err3, err2))

	// test transitivity of errors.Is
	err4 := pkgErrors.NewError("err4", err1, err2, err3)
	require.Equal(t, err1.Error()+": err4: "+err2.Error()+": "+err3.Error(), err4.Error())
	require.True(t, errors.Is(err4, testErr1))
	require.True(t, errors.Is(err4, testErr2))
}
