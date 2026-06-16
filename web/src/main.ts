import { createApp } from "vue";
import "element-plus/theme-chalk/dark/css-vars.css";
import "@/styles/global.css";
import App from "./App.vue";
import router from "./router";

const stored = localStorage.getItem("sola-theme");
const prefersDark =
  stored === "dark" || (!stored && window.matchMedia("(prefers-color-scheme: dark)").matches);
document.documentElement.classList.toggle("dark", prefersDark);

createApp(App).use(router).mount("#app");
