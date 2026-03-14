import "./index.css";
import App from "./App.svelte";
import { mount } from "svelte";
import { themeStore } from "./stores/theme.svelte";

themeStore.init();

const app = mount(App, {
  target: document.getElementById("app")!,
});

export default app;
