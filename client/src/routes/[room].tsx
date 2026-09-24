import { useNavigate, useParams } from "@solidjs/router";
import { For, Show } from "solid-js";
import { createRoomSocket, GameState, OutgoingMessage, PlayerView } from "../websocket";
import { Card } from "../components/card";

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
                            isPlaying={state().game != null}
                        />
                        <OpponentList opponents={state().players} />
                        <div class="bg-neutral-100 p-4 m-4 rounded-2xl">
                            <Show when={state().game}>{(g) => <Game game={g()} />}</Show>
                        </div>
                    </>
                )}
            </Show>
        </div>
    );
}

function Game(props: { game: GameState }) {
    return (
        <>
            <BuildPiles piles={props.game.discardPiles} />
            <StockPile
                canPlay={props.game.isPlayerTurn}
                topCard={props.game.stockPileTopCard}
                count={props.game.stockPileCount}
            />
            <DiscardPiles piles={props.game.discardPiles} />
            <PlayerHand canPlay={props.game.isPlayerTurn} cards={props.game.hand} />
        </>
    );
}
function StockPile(props: { topCard: number; count: number; canPlay: boolean }) {
    return (
        <div>
            <h4>StockPile</h4>
            <div class="bg-neutral-200 p-2 w-fit rounded-2xl">
                <button disabled={!props.canPlay}>
                    <Card value={props.topCard} />
                </button>
                <div class="font-black text-center p-2 text-xl">{props.count}</div>
            </div>
        </div>
    );
}

function BuildPiles(props: { piles: number[][] }) {
    return (
        <div class="flex gap-2">
            <For each={props.piles}>
                {(pile) => (
                    <div class="p-2 bg-neutral-200">
                        <Show when={pile.length > 0} fallback={<p>vide</p>}>
                            <For each={pile}>{(value) => <Card value={value} />}</For>
                        </Show>
                    </div>
                )}
            </For>
        </div>
    );
}

function PlayerHand(props: { cards: number[]; canPlay: boolean }) {
    return (
        <div class="flex justify-center">
            <For each={props.cards}>
                {(value, i) => {
                    const centerIndex = () => (props.cards.length - 1) / 2;

                    const delta = () => i() - centerIndex();
                    const maxDistance = () => centerIndex() || 1;

                    const r = () => (delta() / maxDistance()) * 15; // -15deg à +15deg
                    const y = () => Math.abs(delta()) * 20; // penche vers l'extérieur des deux côtés
                    return (
                        <button
                            class="hover:scale-110 w-[20vw]  transition-transform"
                            style={{
                                "z-index": i(),
                                "margin-left": i() === 0 ? undefined : "-4vw",
                                transform: `translateY(${y()}%) rotate(${r()}deg)`,
                            }}
                            disabled={!props.canPlay}
                        >
                            <Card value={value} />
                        </button>
                    );
                }}
            </For>
        </div>
    );
}

function DiscardPiles(props: { piles: number[][] }) {
    return (
        <div class="flex gap-2">
            <For each={props.piles}>
                {(pile) => (
                    <div class="p-2 bg-neutral-200">
                        <Show when={pile.length > 0} fallback={<p>vide</p>}>
                            <For each={pile}>{(value) => <Card value={value} />}</For>
                        </Show>
                    </div>
                )}
            </For>
        </div>
    );
}
function PlayerInfo(props: {
    id: string;
    isHost: boolean;
    isPlaying: boolean;
    send: (msg: OutgoingMessage) => void;
}) {
    const start = () => props.send({ type: "lobby.start" });

    return (
        <Show when={props.isHost} fallback={<p>{props.id}</p>}>
            <p>{props.id} HOST</p>
            <Show when={!props.isPlaying}>
                <button onClick={start}>Start game</button>
            </Show>
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
