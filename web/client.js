const protocol = location.protocol === "https:" ? "wss" : "ws"
const socket = new WebSocket(`${protocol}://${location.host}/ws`)
const canvas = document.getElementById("game")
const context = canvas.getContext("2d")

let playerID = null
let latestSnapshot = null

function drawPlayer(player) {
    const isLocalPlayer = player.id === playerID

    context.beginPath()
    context.arc(player.x, player.y, 15, 0, Math.PI * 2)

    context.fillStyle = isLocalPlayer ? "green" : "red"
    context.fill()
}

function render() {
    context.clearRect(0, 0, canvas.width, canvas.height)

    if (latestSnapshot !== null) {
        const localPlayer = latestSnapshot.players.find(player => player.id == playerID)
        for (const player of latestSnapshot.players) {
            if (player.id != localPlayer.id) {
                drawPlayer(player)
            }
        }
        if (localPlayer != undefined) {
            drawPlayer(localPlayer)
        }
    }

    requestAnimationFrame(render)
}

requestAnimationFrame(render)

const movement = {
    up: false,
    left: false,
    right: false,
    down: false,
}

const keyDirections = {
    w: "up",
    a: "left",
    s: "down",
    d: "right",
}

// WebSocket

socket.addEventListener("open", () => {
    console.log("WebSocket connected")
})

socket.addEventListener("message", event => {
    const message = JSON.parse(event.data)

    switch (message.type) {
        case "welcome":
            playerID = message.player_id
            console.log("I am", playerID)
            break

        case "snapshot":
            handleSnapshot(message)
            break

        default:
            console.warn("Unknown message:", message)
    }
})

socket.addEventListener("close", event => {
    console.log("WebSocket closed:", event.code, event.reason)
})

socket.addEventListener("error", event => {
    console.error("WebSocket error:", event)
})

// Input

function sendMovement() {
    if (socket.readyState !== WebSocket.OPEN) {
        return
    }

    socket.send(JSON.stringify({
        type: "input",
        ...movement,
    }))
}

function setDirection(key, pressed) {
    const direction = keyDirections[key.toLowerCase()]

    if (direction === undefined) {
        return false
    }

    if (movement[direction] === pressed) {
        return false
    }

    movement[direction] = pressed
    return true
}

window.addEventListener("keydown", event => {
    if (setDirection(event.key, true)) {
        sendMovement()
    }
})

window.addEventListener("keyup", event => {
    if (setDirection(event.key, false)) {
        sendMovement()
    }
})

window.addEventListener("blur", () => {
    const wasMoving = Object.values(movement).some(Boolean)

    movement.up = false
    movement.left = false
    movement.right = false
    movement.down = false

    if (wasMoving) {
        sendMovement()
    }
})

// Snapshots

function handleSnapshot(snapshot) {
    latestSnapshot = snapshot
}