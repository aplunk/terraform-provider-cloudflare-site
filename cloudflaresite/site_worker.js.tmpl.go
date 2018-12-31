package cloudflaresite

const siteWorkerTemplate = `
const namespace = {{ .Namespace }};

const largeFiles = {
{{- range key, fileManifest := .LargeFiles }}
{{ name, size := $fileManifest }}
"{{ $name }}": 0,
{{ end -}}
{{ end -}}
};

const smallFiles = {
{{- range name := .SmallFiles }}
"{{ $name }}": 0,
{{ end -}}
};

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
