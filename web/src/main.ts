import { createApp } from "vue";
import "element-plus/theme-chalk/dark/css-vars.css";
import "@/styles/global.css";
import App from "./App.vue";
import router from "./router";

createApp(App).use(router).mount("#app");
