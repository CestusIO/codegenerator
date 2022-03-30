package templating

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHeaders(t *testing.T) {
	content := bytes.NewBufferString(`!!filename coucou.txt
!!delimiters <<< >>>
!!if-not-exists
!!remove-if-empty
!!if 1
!!no-go-generate
!!pathreplace oldValue newValue
!!pathreplace oldValue2 newValue2
Hello
World`)

	var header Header
	body, err := ParseHeaders(content, &header)
	require.NoError(t, err)

	value, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld", string(value))
	assert.Equal(t, "coucou.txt", header.Filename)
	assert.Equal(t, [2]string{"<<<", ">>>"}, header.Delimiters)
	assert.Equal(t, []pathReplace{{old: "oldValue", new: "newValue"}, {old: "oldValue2", new: "newValue2"}}, header.PathReplace)
	assert.True(t, header.IfNotExists)
	assert.True(t, header.RemoveIfEmpty)
	assert.True(t, header.NoGoGenerate)
	assert.NotNil(t, header.If)
}

func TestParseHeadersUnknownKeyword(t *testing.T) {
	content := bytes.NewBufferString(`!!filename coucou.txt
!!foo bla bla bla
Hello
World`)

	var header Header
	_, err := ParseHeaders(content, &header)

	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseHeadersIncompleteLine(t *testing.T) {
	content := bytes.NewBufferString(`!!filename coucou.txt`)

	var header Header
	body, err := ParseHeaders(content, &header)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expected := `!!filename coucou.txt`
	value, err := io.ReadAll(body)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if string(value) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, value)
	}
}

func TestParseHeadersIf(t *testing.T) {
	content := bytes.NewBufferString(`!!if .Foo
`)

	var header Header
	_, err := ParseHeaders(content, &header)

	require.NoError(t, err)
	require.NotNil(t, header.If)

	t.Run("true", func(t *testing.T) {
		ok, err := header.If(struct{ Foo string }{Foo: "1"})
		assert.True(t, ok)
		assert.NoError(t, err)
	})

	t.Run("false", func(t *testing.T) {
		ok, err := header.If(struct{ Foo string }{Foo: ""})
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		_, err := header.If(struct{}{})
		assert.Error(t, err)
	})
}

func TestParseHeadersIfAnd(t *testing.T) {
	content := bytes.NewBufferString(`!!if .Foo .Bar
`)

	var header Header
	_, err := ParseHeaders(content, &header)

	require.NoError(t, err)
	require.NotNil(t, header.If)

	t.Run("true", func(t *testing.T) {
		ok, err := header.If(struct {
			Foo bool
			Bar bool
		}{Foo: true,
			Bar: true})
		assert.True(t, ok)
		assert.NoError(t, err)
	})

	t.Run("false", func(t *testing.T) {
		ok, err := header.If(struct {
			Foo bool
			Bar bool
		}{
			Foo: true,
			Bar: false,
		})
		assert.False(t, ok)
		assert.NoError(t, err)
	})
	t.Run("false", func(t *testing.T) {
		ok, err := header.If(struct {
			Foo bool
			Bar bool
		}{
			Foo: false,
			Bar: true,
		})
		assert.False(t, ok)
		assert.NoError(t, err)
	})
	t.Run("false", func(t *testing.T) {
		ok, err := header.If(struct {
			Foo bool
			Bar bool
		}{
			Foo: false,
			Bar: false,
		})
		assert.False(t, ok)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		_, err := header.If(struct{}{})
		assert.Error(t, err)
	})
}

func TestParseHeadersIfOr(t *testing.T) {
	content := bytes.NewBufferString(`!!ifor .Foo .Bar
`)

	var header Header
	_, err := ParseHeaders(content, &header)

	require.NoError(t, err)
	require.NotNil(t, header.IfOr)

	t.Run("true", func(t *testing.T) {
		ok, err := header.IfOr(struct {
			Foo bool
			Bar bool
		}{Foo: true,
			Bar: true})
		assert.True(t, ok)
		assert.NoError(t, err)
	})

	t.Run("true", func(t *testing.T) {
		ok, err := header.IfOr(struct {
			Foo bool
			Bar bool
		}{
			Foo: true,
			Bar: false,
		})
		assert.True(t, ok)
		assert.NoError(t, err)
	})
	t.Run("true", func(t *testing.T) {
		ok, err := header.IfOr(struct {
			Foo bool
			Bar bool
		}{
			Foo: false,
			Bar: true,
		})
		assert.True(t, ok)
		assert.NoError(t, err)
	})
	t.Run("false", func(t *testing.T) {
		ok, err := header.IfOr(struct {
			Foo bool
			Bar bool
		}{
			Foo: false,
			Bar: false,
		})
		assert.False(t, ok)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		_, err := header.IfOr(struct{}{})
		assert.Error(t, err)
	})
}
