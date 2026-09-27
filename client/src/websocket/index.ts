import { Accessor, createSignal, onSettled } from "solid-js";
import { ClientMessage, ServerMessage } from "./messages";
import {
    GameView,
    RoomView,
    ServerMessageTypeGameState,
    ServerMessageTypeRoomState,
} from "./generated";

export type RoomSocket = {
    connectionState: Accessor<boolean>;
    roomState: Accessor<RoomView | undefined>;
    gameState: Accessor<GameView | undefined>;
    error: Accessor<string | undefined>;
    send: (msg: ClientMessage) => void;
    close: () => void;
};

export function createRoomSocket(roomId: string): RoomSocket {
    const [connectionState, setConnectionState] = createSignal(false);
    const [roomState, setRoomState] = createSignal<RoomView | undefined>();
    const [gameState, setGameState] = createSignal<GameView | undefined>();
    const [error, setError] = createSignal<string | undefined>();
    let ws: WebSocket | undefined;

    function send(msg: ClientMessage) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify(msg));
    }

    function close() {
        ws?.close();
        ws = undefined;
    }

    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(`${protocol}//${location.host}/room/${roomId}`);
    ws.onopen = () => setConnectionState(true);

    ws.onclose = (e) => {
        setConnectionState(false);
        setRoomState(undefined);
        if (e.code !== 1000) setError(`connection lost [code ${e.code}]`);
        ws = undefined;
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data) as ServerMessage;
        if (msg.type == ServerMessageTypeGameState) setGameState(msg.payload);
        else if (msg.type == ServerMessageTypeRoomState) setRoomState(msg.payload);
        else console.warn("unknown message type", msg);
    };

    return { connectionState, roomState, gameState, error, send, close };
}
