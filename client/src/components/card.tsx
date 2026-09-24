import { Show } from "solid-js";

export function Card(props: { value: number }) {
    const backgroundColor = () => {
        if (props.value <= 4) return "bg-blue-500 text-blue-900";
        if (props.value <= 8) return "bg-red-500 text-red-900";
        return "bg-green-500 text-green-900";
    };

    return (
        <Show when={props.value != 0} fallback={SkipBo()}>
            <div
                class={[
                    "p-2 shadow-md aspect-3/4 font-black text-[10vw] rounded-2xl place-content-center",
                    backgroundColor(),
                ]}
            >
                {props.value}
            </div>
        </Show>
    );
}

function SkipBo() {
    return (
        <div class="p-2 text-[5vw] aspect-3/4 border-4 border-orange-500 bg-orange-500 text-orange-900 rounded-2xl place-content-center">
            <div class="-rotate-45 font-black">SkipBo</div>
        </div>
    );
}
