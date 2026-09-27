import { useNavigate, useParams } from "@solidjs/router";
import { createRoomSocket, RoomSocket } from "../websocket";
import { createEffect, createSignal, onCleanup, Show } from "solid-js";

export default function RoomView() {
    const [socket, setSocket] = createSignal<RoomSocket | null>(null);
    const params = useParams();
    const navigate = useNavigate();

    createEffect(
        () => params.room,
        (roomID) => {
            if (!roomID) {
                navigate("/");
                return;
            }

            const socket = createRoomSocket(roomID);
            setSocket(socket);
            return () => onCleanup(() => socket.close);
        },
    );

    return (
        <Show when={socket()} fallback={"loading"}>
            {(s) => (
                <Show when={!s().error()} fallback={<RoomViewError message={s().error()} />}>
                    <Show when={!s().gameState() && s().roomState()?.isHost}>
                        <button onClick={() => s().send({ type: "lobby.start" })}>Start game</button>
                    </Show>
                </Show>
            )}
        </Show>
    );
}

function RoomViewError(props: { message?: string }) {
    return (
        <div>
            <h1>Unable to connect to the room</h1>
            {props.message && <p>{props.message}</p>}
            <a href="/">Go back home</a>
        </div>
    );
}
