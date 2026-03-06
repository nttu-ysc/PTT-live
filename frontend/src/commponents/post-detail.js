import { FetchPostMessages, SendMessage } from "../../wailsjs/go/pttclient/PttClient";

// Abort controller for the polling loop – replaced each time we open a post
let pollingAborted = false;

export function fetchPostDetail(postId) {
    initializeChat(postId);
}

const pushModes = { 1: '推', 2: '噓', 3: '→' };

document.addEventListener('DOMContentLoaded', () => {
    const sendButton = document.getElementById('send-chat-btn');
    const modeSelector = document.getElementById('modeSelector');
    let currentMode = 1;
    let longPressTimeout;

    sendButton.addEventListener('mousedown', () => {
        longPressTimeout = setTimeout(() => {
            modeSelector.style.display = 'flex';
        }, 500);
    });

    sendButton.addEventListener('mouseup', () => {
        clearTimeout(longPressTimeout);
    });

    function updateSendButtonText() {
        sendButton.textContent = `${pushModes[currentMode]}`;
    }

    document.getElementById('chat-form').addEventListener('submit', (e) => {
        e.preventDefault();
        const chatInput = document.getElementById('chat-input');
        if (modeSelector.style.display !== 'flex') {
            const message = chatInput.value.trim();
            if (message !== '') {
                sendMessage(message);
            }
        }
    });

    document.querySelectorAll('.mode-selector button').forEach(button => {
        button.addEventListener('click', () => {
            currentMode = parseInt(button.getAttribute('data-mode'));
            modeSelector.style.display = 'none';
            updateSendButtonText();
        });
    });

    function sendMessage(messageText) {
        const messageInput = document.getElementById('chat-input');
        SendMessage(currentMode, messageText);
        messageInput.value = '';
    }

    updateSendButtonText();

    // Back button: post-detail → post-list
    const detailBackBtn = document.getElementById('detail-back-btn');
    if (detailBackBtn) {
        detailBackBtn.addEventListener('click', () => {
            // Stop the polling loop
            pollingAborted = true;
            // Clear chat messages so next post starts fresh
            const chatMessages = document.getElementById('chat-messages');
            if (chatMessages) chatMessages.innerHTML = '';
            const chatInput = document.getElementById('chat-input');
            if (chatInput) chatInput.value = '';

            document.querySelector('post-detail-page').style.display = 'none';
            document.querySelector('post-list-page').style.display = 'block';
        });
    }
});


function initializeChat(postId) {
    // Reset abort flag before starting a new fetch loop
    pollingAborted = false;

    // Clear previous messages
    const chatMessages = document.getElementById('chat-messages');
    if (chatMessages) chatMessages.innerHTML = '';

    // Setup polling
    fetchMessages(postId);
}

async function fetchMessages(postId) {
    let hash = '';
    const chatLoading = document.getElementById('chat-loading');
    try {
        while (!pollingAborted) {
            chatLoading.style.display = 'block';
            const messages = await FetchPostMessages(postId, hash);
            if (pollingAborted) break;
            chatLoading.style.display = 'none';
            // Guard: only update hash and display if we got real messages.
            // If messages is null or empty [], skip to avoid hash becoming
            // undefined (which would cause the backend to return all messages
            // from the start on the next poll, causing duplicates).
            if (messages && messages.length > 0) {
                hash = messages[messages.length - 1].hash;
                displayMessages(messages);
            }
            await sleep(1500);
        }
    } catch (error) {
        if (!pollingAborted) {
            console.error(error);
        }
    } finally {
        chatLoading.style.display = 'none';
    }
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

const MAX_MESSAGES = 300; // cap DOM nodes to avoid layout slowdown

function displayMessages(messages) {
    // ONE isAtBottom() check before any DOM change (avoids N layout reflows)
    const wasAtBottom = isAtBottom();

    const batchFragment = document.createDocumentFragment();
    let hasImage = false;

    messages.forEach(msg => {
        const el = buildMessageEl(msg.author, msg.content, wasAtBottom);
        if (el.hasImage) hasImage = true;
        batchFragment.appendChild(el.node);
    });

    chatMessages.appendChild(batchFragment);

    // Trim oldest messages if over the cap
    while (chatMessages.children.length > MAX_MESSAGES) {
        chatMessages.removeChild(chatMessages.firstChild);
    }

    // ONE scroll call for the whole batch
    if (wasAtBottom) {
        scrollToBottom();
    } else {
        newMessageAlert.style.display = 'block';
    }
}

/**
 * Builds a message DOM node WITHOUT touching the live DOM or triggering
 * layout. Returns { node, hasImage } so the caller can decide on scrolling.
 */
function buildMessageEl(author, message, wasAtBottom) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message';

    const authorDiv = document.createElement('div');
    authorDiv.className = 'author';
    authorDiv.style.color = getRandomColor();
    authorDiv.textContent = `${author}: `;

    const contentDiv = document.createElement('div');
    contentDiv.className = 'content';
    contentDiv.textContent = message;

    messageDiv.appendChild(authorDiv);
    messageDiv.appendChild(contentDiv);

    // ── Image URL preview ──────────────────────────────────────────────────
    const imgUrlRegex = /https?:\/\/\S+\.(?:jpg|jpeg|png|gif|webp|bmp)(\?\S*)?/gi;
    const imgUrls = message.match(imgUrlRegex);
    let hasImage = false;
    if (imgUrls) {
        hasImage = true;
        imgUrls.forEach(rawUrl => {
            const url = rawUrl.replace(/^http:\/\//i, 'https://');
            const preview = document.createElement('div');
            preview.className = 'img-preview';
            const img = document.createElement('img');
            img.src = url;
            img.alt = '圖片';
            img.loading = 'lazy';
            img.addEventListener('load', () => {
                contentDiv.textContent = contentDiv.textContent
                    .replace(rawUrl, '').replace(url, '').trim();
                if (wasAtBottom) scrollToBottom();
            });
            img.addEventListener('error', () => { preview.style.display = 'none'; });
            preview.appendChild(img);
            messageDiv.appendChild(preview);
        });
    }
    // ──────────────────────────────────────────────────────────────────────

    return { node: messageDiv, hasImage };
}


const chatMessages = document.getElementById('chat-messages');
const newMessageAlert = document.getElementById('newMessageAlert');

function isAtBottom() {
    // Use a 50px threshold so that image loading (which adds height after the
    // scroll check) doesn't break the auto-scroll detection.
    return chatMessages.scrollHeight - chatMessages.scrollTop - chatMessages.clientHeight < 50;
}

chatMessages.addEventListener('scroll', () => {
    if (isAtBottom()) {
        newMessageAlert.style.display = 'none';
    }
});

function getRandomColor() {
    const letters = '0123456789ABCDEF';
    let color = '#';
    for (let i = 0; i < 6; i++) {
        color += letters[Math.floor(Math.random() * 16)];
    }
    return color;
}
