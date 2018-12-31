package cloudflaresite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/template"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pkg/errors"
)

func resourceCloudflareSite() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudflareSiteCreate,
		Delete: resourceCloudflareSiteDelete,
		Read:   resourceCloudflareSiteRead,
		Update: resourceCloudflareSiteUpdate,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareSiteImport,
		},
		Schema: map[string]*schema.Schema{
			"source": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"chunk_size": {
				Type:    schema.TypeInt,
				Default: 1024,
			},
		},
	}
}

type uploader func(key string, value []byte) error

func uploadFile(pathStr, prefix string, info os.FileInfo, chunkSize int, uploadKV uploader) (keys []string, err error) {
	fh, err := os.Open(pathStr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer fh.Close()

	fileSize := int(info.Size())
	if fileSize == 0 {
		return nil, errors.Errorf("refusing to upload empty file:%s", info.Name())
	}

	data := make([]byte, chunkSize)
	for i := 0; i < fileSize; i += chunkSize {
		read, err := fh.ReadAt(data, int64(i))
		// don't return io.EOF errors
		if err != nil && err != io.EOF {
			return nil, errors.WithStack(err)
		}

		key := prefix
		if i != 0 {
			// files larger than the chunk size are broken up into multiple keys with prefixes ending in their index number
			key = fmt.Sprintf("%s_%d", prefix, int(fileSize/i))
		}

		if err := uploadKV(key, data[:read]); err != nil {
			return nil, errors.WithStack(err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func uploadSite(namespaceID, source string, limit int, uploadKV uploader) ([]string, map[string][]string, error) {
	largeFiles := make(map[string][]string)
	smallFiles := make([]string, 0)
	return smallFiles, largeFiles, filepath.Walk(source, func(pathStr string, info os.FileInfo, err error) error {
		// fail early if an error is passed in
		if err != nil {
			return errors.WithStack(err)
		}

		// unable to upload directories
		if info.IsDir() {
			return nil
		}

		// normalize the file key
		key := strings.Replace(pathStr, string(filepath.Separator), "_", -1)

		// upload large files in chunks returning a mapping of the chunks which will
		// become a manifest enabling reconstructing the original file
		if info.Size() > int64(limit) {
			chunks, err := uploadFile(pathStr, key, info, limit, uploadKV)
			if err != nil {
				return errors.WithStack(err)
			}
			largeFiles[key] = chunks
			return nil
		}

		smallFiles = append(smallFiles, key)
		// files smaller than the limit can be uploaded without returning a mapping
		_, err = uploadFile(pathStr, key, info, limit, uploadKV)
		return err
	})
}

func renderWorkerTemplate(namespaceID string, smallFiles []string, largeFiles map[string][]string, output io.Writer) error {
	tmpl, err := template.New("worker").Parse(siteWorkerTemplate)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(output, "worker", struct {
		Namespace  string
		LargeFiles map[string][]string
		SmallFiles []string
	}{
		namespaceID,
		largeFiles,
		smallFiles,
	})
}

func resourceCloudflareSiteCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	source := d.Get("source").(string)
	namespaceID := d.Get("namespace_id").(string)
	chunkSize := d.Get("chunk_size").(int)

	uploader := func(key string, value []byte) error {
		_, err := client.CreateWorkersKV(context.Background(), namespaceID, key, value)
		return err
	}

	smallFiles, largeFiles, err := uploadSite(namespaceID, source, chunkSize, uploader)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})
	if err = renderWorkerTemplate(namespaceID, smallFiles, largeFiles, buf); err != nil {
		return err
	}

	// todo upload worker script
	buf.String()
	return nil
}

func resourceCloudflareSiteDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceCloudflareSiteRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceCloudflareSiteUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceCloudflareSiteImport(d *schema.ResourceData, meta interface{}) (result []*schema.ResourceData, err error) {
	return nil, nil
}
