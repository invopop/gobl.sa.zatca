package vatexsa_test

import (
	"testing"

	"github.com/invopop/gobl.sa.zatca/catalogues/vatexsa"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// Test that the catalogue is registered
	ed := tax.ExtensionForKey(vatexsa.ExtKeyVATEX)
	require.NotNil(t, ed)
	assert.Equal(t, "vatex-sa", ed.Key.String())
	assert.Len(t, ed.Values, 16)

	for _, def := range ed.Values {
		assert.Regexp(t, `^VATEX-SA-[A-Z0-9-]+$`, def.Code.String())
		assert.NotEmpty(t, def.Name.In(i18n.EN), "missing name for %s", def.Code)
	}

	def := ed.CodeDef("VATEX-SA-32")
	require.NotNil(t, def)
	assert.Equal(t, "Export of goods", def.Name.In(i18n.EN))
	assert.Nil(t, ed.CodeDef("VATEX-EU-132"))
}
