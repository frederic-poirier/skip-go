import { useNavigate, useParams } from "@solidjs/router";
import { For, Show } from "solid-js";
import { createRoomSocket, GameState, OutgoingMessage, PlayerView } from "../websocket";

export default function Room() {
    const params = useParams();
    const navigate = useNavigate();
    if (params.room == undefined) {
        navigate("/");
    }

    const roomSocket = createRoomSocket(params.room!);

    return (
        <div>
            <p>{roomSocket.connectionState() ? "on" : "off"}</p>
            <h1>Room {params.room}</h1>
            <Show when={roomSocket.roomState()} fallback={<p>En attente du room-state…</p>}>
                {(state) => (
                    <>
                        <PlayerInfo
                            send={(msg) => roomSocket.send(msg)}
                            id={state().id}
                            isHost={state().isHost}
                        />
                        <OpponentList opponents={state().players} />
                        <Show when={state().game}>{(g) => <Game game={g()} />}</Show>
                    </>
                )}
            </Show>
        </div>
    );
}

function Game(props: { game: GameState }) {
    return <p>{JSON.stringify(props.game)}</p>;
}

function PlayerInfo(props: { id: string; isHost: boolean; send: (msg: OutgoingMessage) => void }) {
    const start = () => props.send({ type: "lobby.start" });

    return (
        <Show when={props.isHost} fallback={<p>{props.id}</p>}>
            <p>{props.id} HOST</p>
            <button onClick={start}>Start game</button>
        </Show>
    );
}

function OpponentList(props: { opponents: PlayerView[] }) {
    return (
        <ul>
            <For each={props.opponents} fallback={<li>Aucun autre joueur</li>}>
                {(player) => (
                    <li>
                        <p>{player.id}</p>
                        <p>Score: {player.score}</p>
                        <p>Connect: {player.connected ? "oui" : "non"}</p>
                    </li>
                )}
            </For>
        </ul>
    );
}
