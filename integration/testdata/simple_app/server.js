const http = require('http');

const port = process.env.PORT || 8080;

const server = http.createServer((request, response) => {
  return response.end('Hello World!');
});

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`NOT vendored server is listening on ${port}`);
});
