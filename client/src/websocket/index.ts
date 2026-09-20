import { createSignal, onSettled } from "solid-js";

export type PlayerView = {
    id: string;
    score: number;
    connected: boolean;
};

export type GameState = {
    buildPiles: number[][4];
    discardPiles: number[][4];
    hand: number[];
    opponents: OpponentView[];
    stockPileCount: number;
    stockPileCard: number;
    isYourTurn: boolean;
};

type OpponentView = {
    discardPiles: number[][4];
    handCount: number;
    playerId: string;
    stockPileCard: number;
    stockPileCount: number;
};

export type RoomState = {
    roomId: string;
    id: string;
    isHost: boolean;
    players: PlayerView[];
    game: GameState | null;
};

type RoomViewPayload = {
    roomId: string;
    id: string;
    isHost: boolean;
    players?: PlayerView[];
};

type ServerMessage =
    | { type: "room-state"; payload: RoomViewPayload }
    | { type: "game-state"; payload: GameState };

export type OutgoingMessage = {
    type: string;
    payload?: unknown;
};

export function createRoomSocket(roomId: string) {
    const [connectionState, setConnectionState] = createSignal(false);
    const [roomState, setRoomState] = createSignal<RoomState | undefined>();
    let ws: WebSocket | undefined;

    function send(msg: OutgoingMessage) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify(msg));
    }

    function close() {
        ws?.close();
        ws = undefined;
    }

    onSettled(() => {
        const protocol = location.protocol === "https:" ? "wss:" : "ws:";
        ws = new WebSocket(`${protocol}//${location.host}/room/${roomId}`);
        ws.onopen = () => setConnectionState(true);
        ws.onclose = () => {
            setConnectionState(false);
            setRoomState(undefined);
            ws = undefined;
        };
        ws.onmessage = (e) => {
            const msg = JSON.parse(e.data as string) as ServerMessage;

            if (msg.type === "room-state") {
                setRoomState((prev) => ({
                    ...msg.payload,
                    players: msg.payload.players ?? [],
                    game: prev?.game ?? null,
                }));
                return;
            }

            if (msg.type === "game-state") {
                setRoomState((prev) => (prev ? { ...prev, game: msg.payload } : prev));
            }
        };
        return () => close();
    });

    return { connectionState, roomState, send, close };
}
