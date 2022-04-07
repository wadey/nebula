package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slackhq/nebula/test"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Load(t *testing.T) {
	l := test.NewLogger()
	dir, err := ioutil.TempDir("", "config-test")
	// invalid yaml
	c := NewC(l)
	ioutil.WriteFile(filepath.Join(dir, "01.yaml"), []byte(" invalid yaml"), 0644)
	assert.EqualError(t, c.Load(dir), "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into map[interface {}]interface {}")

	// simple multi config merge
	c = NewC(l)
	os.RemoveAll(dir)
	os.Mkdir(dir, 0755)

	assert.Nil(t, err)

	ioutil.WriteFile(filepath.Join(dir, "01.yaml"), []byte("outer:\n  inner: hi"), 0644)
	ioutil.WriteFile(filepath.Join(dir, "02.yml"), []byte("outer:\n  inner: override\nnew: hi"), 0644)
	assert.Nil(t, c.Load(dir))
	expected := map[interface{}]interface{}{
		"outer": map[interface{}]interface{}{
			"inner": "override",
		},
		"new": "hi",
	}
	assert.Equal(t, expected, c.Settings)

	//TODO: test symlinked file
	//TODO: test symlinked directory
}

func TestConfig_Get(t *testing.T) {
	l := test.NewLogger()
	// test simple type
	c := NewC(l)
	c.Settings["firewall"] = map[interface{}]interface{}{"outbound": "hi"}
	assert.Equal(t, "hi", c.Get("firewall.outbound"))

	// test complex type
	inner := []map[interface{}]interface{}{{"port": "1", "code": "2"}}
	c.Settings["firewall"] = map[interface{}]interface{}{"outbound": inner}
	assert.EqualValues(t, inner, c.Get("firewall.outbound"))

	// test missing
	assert.Nil(t, c.Get("firewall.nope"))
}

func TestConfig_GetStringSlice(t *testing.T) {
	l := test.NewLogger()
	c := NewC(l)
	c.Settings["slice"] = []interface{}{"one", "two"}
	assert.Equal(t, []string{"one", "two"}, c.GetStringSlice("slice", []string{}))
}

func TestConfig_GetBool(t *testing.T) {
	l := test.NewLogger()
	c := NewC(l)
	c.Settings["bool"] = true
	assert.Equal(t, true, c.GetBool("bool", false))

	c.Settings["bool"] = "true"
	assert.Equal(t, true, c.GetBool("bool", false))

	c.Settings["bool"] = false
	assert.Equal(t, false, c.GetBool("bool", true))

	c.Settings["bool"] = "false"
	assert.Equal(t, false, c.GetBool("bool", true))

	c.Settings["bool"] = "Y"
	assert.Equal(t, true, c.GetBool("bool", false))

	c.Settings["bool"] = "yEs"
	assert.Equal(t, true, c.GetBool("bool", false))

	c.Settings["bool"] = "N"
	assert.Equal(t, false, c.GetBool("bool", true))

	c.Settings["bool"] = "nO"
	assert.Equal(t, false, c.GetBool("bool", true))
}

func TestConfig_HasChanged(t *testing.T) {
	l := test.NewLogger()
	// No reload has occurred, return false
	c := NewC(l)
	c.Settings["test"] = "hi"
	assert.False(t, c.HasChanged(""))

	// Test key change
	c = NewC(l)
	c.Settings["test"] = "hi"
	c.oldSettings = map[interface{}]interface{}{"test": "no"}
	assert.True(t, c.HasChanged("test"))
	assert.True(t, c.HasChanged(""))

	// No key change
	c = NewC(l)
	c.Settings["test"] = "hi"
	c.oldSettings = map[interface{}]interface{}{"test": "hi"}
	assert.False(t, c.HasChanged("test"))
	assert.False(t, c.HasChanged(""))
}

func TestConfig_ReloadConfig(t *testing.T) {
	l := test.NewLogger()
	done := make(chan bool, 1)
	dir, err := ioutil.TempDir("", "config-test")
	assert.Nil(t, err)
	ioutil.WriteFile(filepath.Join(dir, "01.yaml"), []byte("outer:\n  inner: hi"), 0644)

	c := NewC(l)
	assert.Nil(t, c.Load(dir))

	assert.False(t, c.HasChanged("outer.inner"))
	assert.False(t, c.HasChanged("outer"))
	assert.False(t, c.HasChanged(""))

	ioutil.WriteFile(filepath.Join(dir, "01.yaml"), []byte("outer:\n  inner: ho"), 0644)

	c.RegisterReloadCallback(func(c *C) {
		done <- true
	})

	c.ReloadConfig()
	assert.True(t, c.HasChanged("outer.inner"))
	assert.True(t, c.HasChanged("outer"))
	assert.True(t, c.HasChanged(""))

	// Make sure we call the callbacks
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		panic("timeout")
	}

}
