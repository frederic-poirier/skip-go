import { useParams } from "@solidjs/router";
import { createSignal, For, onSettled, Show } from "solid-js";

type PlayerView = {
    id: string;
    score: number;
    isReady: boolean;
};

type RoomState = {
    roomId: string;
    id: string;
    isReady: boolean;
    players: PlayerView[];
};

type ServerMessage = {
    type: string;
    payload?: RoomState;
};

export default function Room() {
    const [connectionState, setConnectionState] = createSignal("déconnecté");
    const [roomState, setRoomState] = createSignal<RoomState>();
    const params = useParams();

    onSettled(() => {
        const protocol = location.protocol === "https:" ? "wss:" : "ws:";
        const ws = new WebSocket(`${protocol}//${location.host}/room/${params.room}`);

        ws.onopen = () => setConnectionState("connecté");
        ws.onmessage = (e) => {
            const msg = JSON.parse(e.data as string) as ServerMessage;
            if (msg.type === "room-state" && msg.payload) {
                setRoomState({
                    ...msg.payload,
                    players: msg.payload.players ?? [],
                });
            }
        };
        ws.onerror = () => console.error("WS error");
        ws.onclose = () => {
            setConnectionState("déconnecté");
            setRoomState(undefined);
        };

        return () => ws.close();
    });

    return (
        <div>
            <p>{connectionState()}</p>
            <h1>Room {params.room}</h1>
            <Show when={roomState()} fallback={<p>En attente du room-state…</p>}>
                {(state) => (
                    <>
                        <p>
                            Toi ({state().id}) — {state().isReady ? "prêt" : "pas prêt"}
                        </p>
                        <ul>
                            <For each={state().players} fallback={<li>Aucun autre joueur</li>}>
                                {(player) => (
                                    <li>
                                        <p>{player.id}</p>
                                        <p>Score: {player.score}</p>
                                        <p>{player.isReady ? "prêt à jouer" : "pas prêt à jouer"}</p>
                                    </li>
                                )}
                            </For>
                        </ul>
                    </>
                )}
            </Show>
        </div>
    );
}
