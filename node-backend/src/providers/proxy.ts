import httpproxy from 'http-proxy';

const proxy = httpproxy.createProxyServer({});

proxy.on('proxyReq', function(proxyReq, req, res, options) {
    // Cast Node's IncomingMessage to Express Request
    const expressReq = req as unknown as Request;

    // Check the parsed body on the Express request object
    if (expressReq.body && Object.keys(expressReq.body).length > 0) {
        const bodyData = JSON.stringify(expressReq.body);
        
        proxyReq.setHeader('Content-Type', 'application/json');
        proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
        
        proxyReq.write(bodyData);
    }
});



export default proxy;