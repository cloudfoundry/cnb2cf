var logfmt = require("logfmt");
const http = require('http')

const port = process.env.PORT || 8080

const requestHandler = (request, response) => {
  return response.end('Hello World!');
}

const server = http.createServer(requestHandler).listen(port);

server.listen(port, (err) => {
    if (err) {
        return console.log('something bad happened', err)
    }

    console.log(`server is listening on ${port}`)
})
