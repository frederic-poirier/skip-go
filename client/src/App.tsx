import { Title } from "@solidjs/meta";
import { Loading } from "solid-js";
import { paths, Router } from "./router";
import "./styles/index.css";

// The app root: the router and the site-wide layout live here. Pages are
// the modules under src/routes.
export default function App() {
    return (
        <Router>
            {(props) => (
                <>
                    <Title>Solid App</Title>
                    <nav>
                        <a href={paths()}>Home</a>
                    </nav>
                    <Loading fallback={<main>Loading…</main>}>{props.children}</Loading>
                </>
            )}
        </Router>
    );
}
