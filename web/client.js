const protocol = location.protocol === "https:" ? "wss" : "ws"
const socket = new WebSocket(`${protocol}://${location.host}/ws`)
const canvas = document.getElementById("game")
const context = canvas.getContext("2d")

const VIEW_WIDTH = 1280
const VIEW_HEIGHT = 720
const GRID_SIZE = 100

let playerID = null
let latestSnapshot = null
let worldWidth = 3840
let worldHeight = 2160
let connectionStatus = "connecting"

let pixelRatio = 1
let viewportScale = 1
let viewportOffsetX = 0
let viewportOffsetY = 0

function resizeCanvas() {
    const screenWidth = window.innerWidth
    const screenHeight = window.innerHeight

    pixelRatio = window.devicePixelRatio || 1
    canvas.width = Math.floor(screenWidth * pixelRatio)
    canvas.height = Math.floor(screenHeight * pixelRatio)

    viewportScale = Math.min(
        screenWidth / VIEW_WIDTH,
        screenHeight / VIEW_HEIGHT,
    )
    viewportOffsetX = (screenWidth - VIEW_WIDTH * viewportScale) / 2
    viewportOffsetY = (screenHeight - VIEW_HEIGHT * viewportScale) / 2
}

window.addEventListener("resize", resizeCanvas)
resizeCanvas()

function beginFrame() {
    context.setTransform(1, 0, 0, 1, 0, 0)
    context.fillStyle = "#09090b"
    context.fillRect(0, 0, canvas.width, canvas.height)

    context.setTransform(
        pixelRatio * viewportScale,
        0,
        0,
        pixelRatio * viewportScale,
        pixelRatio * viewportOffsetX,
        pixelRatio * viewportOffsetY,
    )

    context.save()
    context.beginPath()
    context.rect(0, 0, VIEW_WIDTH, VIEW_HEIGHT)
    context.clip()

    context.fillStyle = "#18181b"
    context.fillRect(0, 0, VIEW_WIDTH, VIEW_HEIGHT)
}

function endFrame() {
    context.restore()
}

function worldToScreen(x, y, camera) {
    return {
        x: x - camera.x + VIEW_WIDTH / 2,
        y: y - camera.y + VIEW_HEIGHT / 2,
    }
}

function drawGrid(camera) {
    const cameraLeft = camera.x - VIEW_WIDTH / 2
    const cameraTop = camera.y - VIEW_HEIGHT / 2
    const firstWorldX = Math.floor(cameraLeft / GRID_SIZE) * GRID_SIZE
    const firstWorldY = Math.floor(cameraTop / GRID_SIZE) * GRID_SIZE

    context.beginPath()

    for (
        let worldX = firstWorldX;
        worldX <= cameraLeft + VIEW_WIDTH;
        worldX += GRID_SIZE
    ) {
        const screenX = worldX - cameraLeft
        context.moveTo(screenX, 0)
        context.lineTo(screenX, VIEW_HEIGHT)
    }

    for (
        let worldY = firstWorldY;
        worldY <= cameraTop + VIEW_HEIGHT;
        worldY += GRID_SIZE
    ) {
        const screenY = worldY - cameraTop
        context.moveTo(0, screenY)
        context.lineTo(VIEW_WIDTH, screenY)
    }

    context.strokeStyle = "#27272a"
    context.lineWidth = 1
    context.stroke()
}

function drawWorldBounds(camera) {
    const origin = worldToScreen(0, 0, camera)
    context.strokeStyle = "#52525b"
    context.lineWidth = 4
    context.strokeRect(origin.x, origin.y, worldWidth, worldHeight)
}

function drawCrown(crown, camera) {
    const position = worldToScreen(crown.x, crown.y, camera)
    const margin = 35

    const isOutside =
        position.x < margin ||
        position.x > VIEW_WIDTH - margin ||
        position.y < margin ||
        position.y > VIEW_HEIGHT - margin

    if (isOutside) {
        const dx = position.x - VIEW_WIDTH / 2
        const dy = position.y - VIEW_HEIGHT / 2
        const scale = Math.min(
            (VIEW_WIDTH / 2 - margin) / Math.max(Math.abs(dx), 0.001),
            (VIEW_HEIGHT / 2 - margin) / Math.max(Math.abs(dy), 0.001),
        )
        const x = VIEW_WIDTH / 2 + dx * scale
        const y = VIEW_HEIGHT / 2 + dy * scale
        const angle = Math.atan2(dy, dx)

        context.save()
        context.translate(x, y)
        context.rotate(angle)
        context.beginPath()
        context.moveTo(12, 0)
        context.lineTo(-8, -8)
        context.lineTo(-8, 8)
        context.closePath()
        context.fillStyle = "#facc15"
        context.fill()
        context.restore()
        return
    }

    context.beginPath()
    if (crown.holder_id !== "") {
        context.arc(position.x, position.y, 23, 0, Math.PI * 2)
        context.strokeStyle = "#facc15"
        context.lineWidth = 4
        context.stroke()
        return
    }

    context.arc(position.x, position.y, 10, 0, Math.PI * 2)
    context.fillStyle = "#facc15"
    context.fill()
}

function drawPickup(pickup, camera) {
    const position = worldToScreen(pickup.x, pickup.y, camera)
    const color = pickup.type === "speed" ? "#22d3ee" : "#c084fc"

    context.save()
    context.translate(position.x, position.y)
    context.rotate(Math.PI / 4)
    context.fillStyle = color
    context.fillRect(-9, -9, 18, 18)
    context.strokeStyle = "#fafafa"
    context.lineWidth = 2
    context.strokeRect(-9, -9, 18, 18)
    context.restore()
}

function drawPlayer(player, camera) {
    const position = worldToScreen(player.x, player.y, camera)
    const isLocalPlayer = player.id === playerID

    context.save()
    if (player.ghost_active) {
        context.globalAlpha = 0.4
    }

    context.beginPath()
    context.arc(position.x, position.y, 15, 0, Math.PI * 2)
    context.fillStyle = isLocalPlayer ? "#22c55e" : "#ef4444"
    context.fill()

    if (player.speed_active) {
        context.beginPath()
        context.arc(position.x, position.y, 19, 0, Math.PI * 2)
        context.strokeStyle = "#22d3ee"
        context.lineWidth = 3
        context.stroke()
    }
    context.restore()
}

function itemName(item) {
    if (item === "speed") return "Speed"
    if (item === "ghost") return "Ghost"
    return "Empty"
}

function itemColor(item) {
    if (item === "speed") return "#22d3ee"
    if (item === "ghost") return "#c084fc"
    return "#52525b"
}

function drawHud(snapshot, jogadorLocal) {
    context.save()
    context.font = "18px system-ui, sans-serif"
    context.textBaseline = "middle"

    const placar = [...snapshot.players].sort(
        (a, b) => b.crown_time - a.crown_time
    )

    context.fillStyle = "rgba(9, 9, 11, 0.78)"
    context.fillRect(20, 20, 250, 44 + placar.length * 28)
    context.fillStyle = "#facc15"
    context.font = "bold 19px system-ui, sans-serif"
    context.fillText("TAKE THE CROWN", 36, 44)

    context.font = "16px system-ui, sans-serif"
    placar.forEach((jogador, index) => {
        const nome = jogador.id === playerID ? "You" : jogador.id
        const marcador = snapshot.crown.holder_id === jogador.id ? "  ♛" : ""
        context.fillStyle = jogador.id === playerID ? "#86efac" : "#fafafa"
        context.fillText(`${nome}${marcador}`, 36, 78 + index * 28)

        context.textAlign = "right"
        context.fillText(
            `${jogador.crown_time.toFixed(1)} / ${snapshot.target_time.toFixed(0)}s`,
            252,
            78 + index * 28,
        )
        context.textAlign = "left"
    })

    const slotWidth = 150
    const gap = 12
    const startX = VIEW_WIDTH / 2 - slotWidth - gap / 2
    const slotY = VIEW_HEIGHT - 82

    jogadorLocal.inventory.forEach((item, index) => {
        const x = startX + index * (slotWidth + gap)
        context.fillStyle = "rgba(9, 9, 11, 0.82)"
        context.fillRect(x, slotY, slotWidth, 54)
        context.strokeStyle = itemColor(item)
        context.lineWidth = 2
        context.strokeRect(x, slotY, slotWidth, 54)

        context.fillStyle = "#a1a1aa"
        context.font = "14px system-ui, sans-serif"
        context.fillText(index === 0 ? "Q" : "E", x + 14, slotY + 17)
        context.fillStyle = itemColor(item)
        context.font = "bold 17px system-ui, sans-serif"
        context.fillText(itemName(item), x + 14, slotY + 37)
    })

    context.textAlign = "right"
    context.fillStyle = "#a1a1aa"
    context.font = "14px system-ui, sans-serif"
    context.fillText("WASD move  •  Q / E use items", VIEW_WIDTH - 22, VIEW_HEIGHT - 24)
    context.textAlign = "left"

    if (snapshot.status === "waiting") {
        drawCenterMessage("Waiting for another player", "Open another tab to start")
    } else if (snapshot.status === "finished") {
        const winner = snapshot.winner_id === playerID ? "You win!" : `${snapshot.winner_id} wins!`
        drawCenterMessage(
            winner,
            `Next round in ${Math.ceil(snapshot.restart_remaining)}s`,
        )
    }

    context.restore()
}

function drawCenterMessage(title, subtitle) {
    const width = 430
    const height = 110
    const x = VIEW_WIDTH / 2 - width / 2
    const y = 75

    context.fillStyle = "rgba(9, 9, 11, 0.86)"
    context.fillRect(x, y, width, height)
    context.strokeStyle = "#facc15"
    context.lineWidth = 2
    context.strokeRect(x, y, width, height)

    context.textAlign = "center"
    context.fillStyle = "#fafafa"
    context.font = "bold 27px system-ui, sans-serif"
    context.fillText(title, VIEW_WIDTH / 2, y + 42)
    context.fillStyle = "#a1a1aa"
    context.font = "16px system-ui, sans-serif"
    context.fillText(subtitle, VIEW_WIDTH / 2, y + 78)
    context.textAlign = "left"
}

function render() {
    beginFrame()

    if (latestSnapshot !== null) {
        const jogadorLocal = latestSnapshot.players.find(
            player => player.id === playerID
        )

        if (jogadorLocal !== undefined) {
            const camera = {
                x: jogadorLocal.x,
                y: jogadorLocal.y,
            }

            drawGrid(camera)
            drawWorldBounds(camera)

            for (const pickup of latestSnapshot.pickups) {
                drawPickup(pickup, camera)
            }

            drawCrown(latestSnapshot.crown, camera)

            for (const player of latestSnapshot.players) {
                if (player.id !== playerID) {
                    drawPlayer(player, camera)
                }
            }

            drawPlayer(jogadorLocal, camera)
            drawHud(latestSnapshot, jogadorLocal)
        }
    } else {
        drawCenterMessage("Connecting", connectionStatus)
    }

    endFrame()
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

socket.addEventListener("open", () => {
    connectionStatus = "connected"
})

socket.addEventListener("message", event => {
    const message = JSON.parse(event.data)

    switch (message.type) {
        case "welcome":
            playerID = message.player_id
            worldWidth = message.world_width
            worldHeight = message.world_height
            break
        case "snapshot":
            latestSnapshot = message
            break
        default:
            console.warn("Unknown message:", message)
    }
})

socket.addEventListener("close", event => {
    connectionStatus = `closed (${event.code})`
})

socket.addEventListener("error", () => {
    connectionStatus = "connection error"
})

function sendMovement() {
    if (socket.readyState !== WebSocket.OPEN) {
        return
    }

    socket.send(JSON.stringify({
        type: "input",
        ...movement,
    }))
}

function useItem(slot) {
    if (socket.readyState !== WebSocket.OPEN) {
        return
    }
    socket.send(JSON.stringify({
        type: "use_item",
        slot,
    }))
}

function setDirection(key, pressed) {
    const direction = keyDirections[key.toLowerCase()]
    if (direction === undefined || movement[direction] === pressed) {
        return false
    }

    movement[direction] = pressed
    return true
}

window.addEventListener("keydown", event => {
    const key = event.key.toLowerCase()

    if (!event.repeat && (key === "q" || key === "e")) {
        useItem(key === "q" ? 0 : 1)
        return
    }

    if (setDirection(key, true)) {
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
