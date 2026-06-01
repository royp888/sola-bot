import { createApp } from "vue";
import ElementPlus from "element-plus";
import "element-plus/theme-chalk/dark/css-vars.css";
import "element-plus/dist/index.css";
import "@/styles/global.css";
import App from "./App.vue";
import router from "./router";

createApp(App).use(router).use(ElementPlus).mount("#app");
