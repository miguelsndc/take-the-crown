const protocol = location.protocol == "https:" ? "wss" : "ws"
const socket = new WebSocket(`${protocol}://${location.host}/ws`)

socket.addEventListener("open", () => {
    console.log("websocket connected")
    socket.send("hi from client")
})

socket.addEventListener("message", event => {
    const message = JSON.parse(event.data)
    switch (message.type) {
        case "welcome":
            playerID = message.player_id
            console.log("I am", playerID)
            break;
        case "snapshot":
            if (message.tick % 30 === 0) {
                console.log("tick", message.tick, "players", message.players)
            }
        default:
            console.warn("unknown msg")
            break;
    }
});

socket.addEventListener("close", event => {
    console.log("WebSocket closed:", event.code, event.reason);
});

socket.addEventListener("error", event => {
    console.error("WebSocket error:", event);
});