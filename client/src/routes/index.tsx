import { createSignal } from "solid-js";

export default function Home() {
    const [roomId, setRoomId] = createSignal("");

    async function create() {
        const response = await fetch(`/room/create`, {
            method: "POST",
        });

        if (!response.ok) return;
        const data = await response.json();
        console.log(data);
        setRoomId(data.roomID);
    }

    return (
        <main>
            <button onClick={create}>Create room</button>
            <div>
                <label for="roomId">Code de la room:</label>
                <input
                    id="roomId"
                    value={roomId()}
                    onInput={(e) => setRoomId(e.target.value)}
                    placeholder="#AB1234"
                />
                <a href={`/${roomId()}`}>Rejoindre</a>
            </div>
            <Login />
        </main>
    );
}

function Login() {
    const [name, setName] = createSignal(getName());

    async function login() {
        const response = await fetch(`/login/${name()}`, { method: "POST" });
        if (!response.ok) return;
    }

    function getName() {
        return (
            document.cookie
                .split("; ")
                .find((c) => c.startsWith("id="))
                ?.split("=")[1] ?? "Unknown"
        );
    }

    return (
        <>
            <label for="name">Nom:</label>
            <input
                type="text"
                id="name"
                onInput={(e) => setName(e.target.value)}
                value={name()}
                placeholder="Fred"
            />
            <button onClick={login}>Sauvegarder</button>
        </>
    );
}
