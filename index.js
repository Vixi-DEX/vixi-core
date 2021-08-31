import WebSocket, { WebSocketServer } from 'ws';
const { Pool, Client } = require('pg')

const pool = new Pool({
  user: 'postgres',
  host: 'localhost'
  database: 'zigzag',
  password: 'postgres',
  port: 5432,
})

const wss = new WebSocketServer({
  port: 8080,
});

wss.on('connection', function connection(ws) {
    ws.on('message', function incoming(json) {
        console.log('Received: %s', json);
        const msg = JSON.parse(json);
        handleMessage(msg, ws);
    });
});

function handleMessage(msg, ws) {
    switch (msg.op) {
        case "ping":
            response = {"op": "pong"}
            ws.send(JSON.stringify(response))
            break
        case "order":
            // save order to DB
            break
        case "orderbook":
            // respond with market data
            break
        default:
            break
    }
}
