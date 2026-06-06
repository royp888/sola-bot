import { createApp } from "vue";
import "@/mini/styles/theme.css";
import App from "@/mini/App.vue";
import router from "@/mini/router";

createApp(App).use(router).mount("#mini-app");
