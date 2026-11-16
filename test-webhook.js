import http from "http";

// Define the port your webhook will listen on
const PORT = 3001;

// Create the HTTP server
const server = http.createServer((req, res) => {
  // 1. Only allow POST requests (standard for webhooks) at a specific endpoint
  if (req.method === "POST" && req.url === "/webhook") {
    let body = "";

    // 2. Listen for incoming data chunks (streams)
    req.on("data", (chunk) => {
      body += chunk.toString();
    });

    // 3. Once all data chunks are received
    req.on("end", () => {
      try {
        // 4. Parse the raw string body into a JSON object
        const payload = JSON.parse(body);

        // --- PROCESS YOUR WEBHOOK PAYLOAD HERE ---
        console.log("🚀 Webhook received successfully!");
        console.log("Payload Data:", payload);
        // ----------------------------------------

        // 5. Send a 200 OK success response back to the webhook provider
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({ status: "success", message: "Webhook processed" }),
        );
      } catch (error) {
        // Handle malformed JSON data gracefully
        console.error("❌ Failed to parse webhook JSON:", error.message);
        res.writeHead(400, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({ status: "error", message: "Invalid JSON payload" }),
        );
      }
    });
  } else {
    // 6. Handle any other requests (e.g., wrong URL route or GET methods)
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ status: "error", message: "Route not found" }));
  }
});

// Start listening for incoming network traffic
server.listen(PORT, () => {
  console.log(
    `📡 Pure Node.js Webhook server is running at http://localhost:${PORT}/`,
  );
});
