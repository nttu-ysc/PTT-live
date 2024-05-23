import {fetchPostDetail} from "./post-detail";

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

    // document.querySelector('.list-back').addEventListener('click', (e) => {
    //     e.preventDefault();
    //     document.querySelector('post-list-page').style.display = 'none';
    //     document.querySelector('board-page').style.display = 'block';
    // })
}

