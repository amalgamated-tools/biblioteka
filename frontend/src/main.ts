import "./index.css";
import App from "./App.svelte";
import { mount } from "svelte";
import { initTheme } from "./stores/theme";

initTheme();

const app = mount(App, {
  target: document.getElementById("app")!,
});

export default app;
