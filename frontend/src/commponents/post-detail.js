import {FetchPostMessages, SendMessage} from "../../wailsjs/go/pttclient/PttClient";

export function fetchPostDetail(postId) {
    initializeChat(postId);
}

const pushModes = {1: '推', 2: '噓', 3: '→'};

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
});


function initializeChat(postId) {
    const chatInput = document.getElementById('chat-input');
    const sendChatBtn = document.getElementById('send-chat-btn');

    // Setup polling
    fetchMessages(postId);
}

// document.querySelector('.detail-back').addEventListener('click', (e) => {
//     e.preventDefault();
//     document.querySelector('post-detail-page').style.display = 'none';
//     document.querySelector('post-list-page').style.display = 'block';
// })

async function fetchMessages(postId) {
    let hash = ''
    const chatLoading = document.getElementById('chat-loading');
    try {
        while (true) {
            chatLoading.style.display = 'block';
            const messages = await FetchPostMessages(postId, hash)
            chatLoading.style.display = 'none';
            console.log("hash: ", hash)
            console.log('Fetching messages:', messages);
            if (messages !== null) {
                hash = messages[messages.length - 1].hash;
                displayMessages(messages);
            }
            await sleep(1500)
        }
    } catch (error) {
        console.error(error);
    }
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

function displayMessages(messages) {
    messages.forEach(message => {
        displayMessage(message.author, message.content);
    });
}

const chatMessages = document.getElementById('chat-messages');
const newMessageAlert = document.getElementById('newMessageAlert');

function displayMessage(author, message) {
    const isBottom = isAtBottom();
    const fragment = document.createDocumentFragment();
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
    fragment.appendChild(messageDiv);
    // chatMessages.appendChild(messageDiv);
    chatMessages.appendChild(fragment);
    // Scroll to bottom
    if (isBottom) {
        // chatMessages.scrollTop = chatMessages.scrollHeight;
        scrollToBottom();
        console.log('is at bottom.');
    } else {
        newMessageAlert.style.display = 'block';
    }

    requestAnimationFrame(() => {
        chatMessages.style.display = 'none';
        chatMessages.offsetHeight; // Trigger reflow
        chatMessages.style.display = 'block';
    });
}

function isAtBottom() {
    return chatMessages.scrollHeight - chatMessages.scrollTop === chatMessages.clientHeight;
}

chatMessages.addEventListener('scroll', () => {
    if (isAtBottom()) {
        newMessageAlert.style.display = 'none';
    }
})

function getRandomColor() {
    const letters = '0123456789ABCDEF';
    let color = '#';
    for (let i = 0; i < 6; i++) {
        color += letters[Math.floor(Math.random() * 16)];
    }
    return color;
}
