import { GotoBoard } from '../../wailsjs/go/pttclient/PttClient.js';
import { fetchPostDetail } from "./post-detail";

// Keep track of current board for refresh
let currentBoardName = '';

export function setCurrentBoard(name) {
    currentBoardName = name;
}

export function displayPosts(posts) {
    const postList = document.querySelector('#post-list');

    postList.innerHTML = "";

    posts.forEach(post => {
        const postDiv = document.createElement('div');
        postDiv.className = 'post';

        const color = post.push_count === '爆' ? '#f66' : '#ff6';

        postDiv.innerHTML = `
            <h3><span style="color: ${color}">${post.push_count}</span> ${post.title}</h3>
            <p>Author: ${post.author ?? '-'}</p>
            <p>Date: ${post.date ?? '-'}</p>
        `;

        postDiv.addEventListener('click', () => {
            const loadingContainer = document.querySelector('#loading-container');
            loadingContainer.style.display = 'flex';
            document.querySelector('post-list-page').style.display = 'none';
            document.querySelector('post-detail-page').style.display = 'block';
            fetchPostDetail(post.aid);
            loadingContainer.style.display = 'none';
        });

        postList.appendChild(postDiv);
    });
}

// Wire up toolbar buttons once DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Back button: post-list → board
    const listBackBtn = document.getElementById('list-back-btn');
    if (listBackBtn) {
        listBackBtn.addEventListener('click', () => {
            document.querySelector('post-list-page').style.display = 'none';
            document.querySelector('board-page').style.display = 'block';
        });
    }

    // Refresh button: re-fetch posts from current board
    const listRefreshBtn = document.getElementById('list-refresh-btn');
    if (listRefreshBtn) {
        listRefreshBtn.addEventListener('click', async () => {
            if (!currentBoardName) return;

            // Trigger spin animation once
            listRefreshBtn.classList.add('refreshing');
            listRefreshBtn.addEventListener('animationend', () => {
                listRefreshBtn.classList.remove('refreshing');
            }, { once: true });

            const loadingContainer = document.getElementById('loading-container');
            loadingContainer.style.display = 'flex';
            try {
                const posts = await GotoBoard(currentBoardName);
                displayPosts(posts);
            } catch (err) {
                console.error('Refresh failed:', err);
            } finally {
                loadingContainer.style.display = 'none';
            }
        });
    }
});
