package cloudflaresite

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
)

type testFile struct {
	path    []string
	name    string
	size    int
	pathStr string
	key     string
	data    []byte
}

func (t *testFile) make() error {
	t.pathStr = path.Join(os.TempDir(), path.Join(t.path...))
	if err := os.MkdirAll(t.pathStr, 0700); err != nil {
		return err
	}

	fh, err := ioutil.TempFile(t.pathStr, t.name)
	if err != nil {
		return err
	}
	defer fh.Close()

	t.key = strings.Replace(fh.Name(), string(os.PathSeparator), "_", -1)

	t.data = make([]byte, t.size)
	_, err = rand.Read(t.data)
	if err != nil {
		return err
	}

	_, err = fh.Write(t.data)
	return err
}

func (t *testFile) cleanup() {
	os.RemoveAll(t.pathStr)
}

func TestUploadSite(t *testing.T) {
	tests := []*testFile{
		&testFile{
			[]string{"terraform-site-test"},
			"one",
			50,
			"",
			"",
			nil,
		},
		&testFile{
			[]string{"terraform-site-test"},
			"two",
			49,
			"",
			"",
			nil,
		},
		&testFile{
			[]string{"terraform-site-test", "nested"},
			"three",
			30,
			"",
			"",
			nil,
		},
	}

	for _, setup := range tests {
		if err := setup.make(); err != nil {
			t.Fatal(err)
		}
		defer setup.cleanup()
	}

	expected := map[string][]byte{
		// file 1 chunk 1
		tests[0].key: tests[0].data[0:49],
		// file 1 chunk 2
		fmt.Sprintf("%s_1", tests[0].key): tests[0].data[49:],
		// file 2
		tests[1].key: tests[1].data,
		// file 3
		tests[2].key: tests[2].data,
	}

	test := func(key string, value []byte) error {
		expectedVal, ok := expected[key]
		if !ok {
			for _, x := range tests {
				t.Log(x.key)
			}

			t.Fatalf("missing expected key:%s", key)
		}

		if !bytes.Equal(expectedVal, value) {
			t.Fatalf("key:%s expected value:%d != actual value:%d", key, len(expectedVal), len(value))
		}
		return nil
	}

	_, _, err := uploadSite("test_namespace", path.Join(os.TempDir(), "terraform-site-test"), 49, test)
	if err != nil && err != io.EOF {
		t.Fatalf("%+v", err)
	}
}

func TestRenderTemplate(t *testing.T) {
	expected := `
const namespace = test_namespace;
const largeFiles = {"large_file_1":["large_file_1","large_file_1_1"]};
const smallFiles = ["small_file_1","small_file_2"];

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request))
   })

   async function handleRequest(request) {
     var url = new URL(request.url);
     var key = url.hostname.replace(/\//g, "_");

     var content = null;
     if (key in largeFiles) {
        content = streamParts(namespace, largeFiles[key]);
     } else if(key in smallFiles) {
        //todo get content type (arrayBuffer for image, blank for text)
        content = await namespace.get(key);
     }

     if (content === null) {
         return new Response("not found", {status: 404});
     }

     var contentType = "text/html";
     return new Response(content, {headers: {"Content-Type": contentType}});
   }

function streamParts(namespace, chunkKeys) {
    return new ReadableStream({
        start(controller) {
            // todo not strictly needed if we guarentee a sorted manifest.
            chunkKeys.sort();
            for(key in chunkKeys) {
                const stream = await namespace.get(key, 'stream')
                stream.read().then(function process({done, value}){
                    if(done) {
                        return;
                    }
                    controller.enqueue(value);
                    return ReadableStreamReader.read().then(process);
                }
            );
        }
        controller.close();
    }});
}
`
	actual := bytes.NewBuffer([]byte{})
	err := renderWorkerTemplate(
		"test_namespace",
		[]string{"small_file_1", "small_file_2"},
		map[string][]string{
			"large_file_1": []string{
				"large_file_1",
				"large_file_1_1",
			},
		},
		actual,
	)
	if err != nil {
		t.Fatal(err)
	}

	if expected != actual.String() {
		t.Fatalf("expected worker %s\n != actual worker %s", expected, actual)
	}
}
