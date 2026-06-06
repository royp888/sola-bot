import { ref, type Ref } from "vue";
import type { ChatRecord } from "@/types/api";

interface ChatState {
  chats: Ref<ChatRecord[]>;
  selectedChat: Ref<ChatRecord | null>;
  loading: Ref<boolean>;
  error: Ref<string | null>;
  setChats: (list: ChatRecord[]) => void;
  selectChat: (chat: ChatRecord | null) => void;
}

const chats = ref<ChatRecord[]>([]);
const selectedChat = ref<ChatRecord | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

export function useChatStore(): ChatState {
  function setChats(list: ChatRecord[]): void {
    chats.value = list;
    error.value = null;
  }

  function selectChat(chat: ChatRecord | null): void {
    selectedChat.value = chat;
  }

  return { chats, selectedChat, loading, error, setChats, selectChat };
}
