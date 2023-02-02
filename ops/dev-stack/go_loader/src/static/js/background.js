const API_URL = 'https://api.nasa.gov/planetary/apod?api_key=226ePwiZlHfkU4Aq5NSGdRG899ygxBY2bZu8MURc';

fetch(API_URL)
    .then(response => response.json())
    .then(data => {
    document.body.style.backgroundImage = `url(${data.url})`;
    })
    .catch(error => {
    console.error(error);
    });
