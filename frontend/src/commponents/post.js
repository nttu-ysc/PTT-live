import {fetchPostDetail} from "./post-detail";

export function displayPosts(posts) {
    const postList = document.querySelector('#post-list');

    postList.innerHTML = "";

    posts.forEach(post => {
        const postDiv = document.createElement('div');
        postDiv.className = 'post';

        postDiv.innerHTML = `
            <h3>${post.title}</h3>
            <p>Author: ${post.author??'-'}</p>
            <p>Date: ${post.date??'-'}</p>
        `;

        postDiv.addEventListener('click', () => {
            const loadingContainer = document.querySelector('#loading-container');
            loadingContainer.style.display = 'flex';
            document.querySelector('post-list-page').style.display = 'none';
            document.querySelector('post-detail-page').style.display = 'block';
            fetchPostDetail(post.search_id);
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

