package model

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMock3MF(t *testing.T, xmlContent string) []byte {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add 3D/3dmodel.model file
	f, err := zipWriter.Create("3D/3dmodel.model")
	require.NoError(t, err)
	_, err = f.Write([]byte(xmlContent))
	require.NoError(t, err)

	err = zipWriter.Close()
	require.NoError(t, err)

	return buf.Bytes()
}

func TestLoad3MF_Success(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
	<metadata name="Title">Box</metadata>
	<metadata name="slic3rpe:layer_height">0.15</metadata>
	<metadata name="slic3rpe:fill_density">15%</metadata>
	<resources>
		<object id="1" type="model">
			<mesh>
				<vertices>
					<vertex x="0.0" y="0.0" z="0.0" />
					<vertex x="10.0" y="0.0" z="0.0" />
					<vertex x="0.0" y="10.0" z="0.0" />
				</vertices>
				<triangles>
					<triangle v1="0" v2="1" v3="2" />
				</triangles>
			</mesh>
		</object>
	</resources>
	<build>
		<item objectid="1" transform="1 0 0 0 1 0 0 0 1 0 0 0" />
	</build>
</model>`

	zipData := createMock3MF(t, xmlContent)
	r := bytes.NewReader(zipData)

	bm, err := Load3MF(r, "test3mf")
	require.NoError(t, err)
	assert.NotNil(t, bm)

	assert.Equal(t, "test3mf", bm.Name)
	assert.Equal(t, 1, len(bm.Triangles))
	assert.Equal(t, 10.0, bm.Triangles[0].V2.X)

	// Check metadata parsing
	assert.Equal(t, 0.15, bm.SliceConfig.LayerHeight)
	assert.Equal(t, 0.15, bm.SliceConfig.InfillDensity) // 15% -> 0.15
}

func TestLoad3MF_UnitScaling(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<model unit="inch" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
	<resources>
		<object id="1" type="model">
			<mesh>
				<vertices>
					<vertex x="1.0" y="0.0" z="0.0" />
					<vertex x="0.0" y="1.0" z="0.0" />
					<vertex x="0.0" y="0.0" z="1.0" />
				</vertices>
				<triangles>
					<triangle v1="0" v2="1" v3="2" />
				</triangles>
			</mesh>
		</object>
	</resources>
	<build>
		<item objectid="1" />
	</build>
</model>`

	zipData := createMock3MF(t, xmlContent)
	r := bytes.NewReader(zipData)

	bm, err := Load3MF(r, "testInch")
	require.NoError(t, err)
	assert.NotNil(t, bm)

	// Since unit="inch", vertices should be multiplied by 25.4
	assert.Equal(t, 25.4, bm.Triangles[0].V1.X)
}

func TestLoad3MF_MissingModelFile(t *testing.T) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add wrong file
	f, _ := zipWriter.Create("wrong.txt")
	f.Write([]byte("not a model"))
	zipWriter.Close()

	r := bytes.NewReader(buf.Bytes())
	_, err := Load3MF(r, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find 3D/3dmodel.model")
}

func TestParse3MFMetadata(t *testing.T) {
	metadata := []metadataXML{
		{Name: "cura:layer_height", Value: "0.2"},
		{Name: "cura:infill_sparse_density", Value: "0.3"},
		{Name: "cura:material_print_temperature", Value: "215"},
		{Name: "cura:material_bed_temperature", Value: "60"},
	}

	config := NewSliceConfig()
	parse3MFMetadata(metadata, config)

	assert.Equal(t, 0.2, config.LayerHeight)
	assert.Equal(t, 0.3, config.InfillDensity)
	assert.Equal(t, 215.0, config.NozzleTemp)
	assert.Equal(t, 60.0, config.BedTemp)
}
