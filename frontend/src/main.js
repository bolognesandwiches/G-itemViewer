import './index.css';

document.querySelector('#app').innerHTML = `
    <div class="window">
        <div class="topleft"></div>
        <div class="top"></div>
        <div class="topright"></div>
        <div class="left"></div>
        <div class="content">
            <button id="launchButton">Launch and Embed Habbo</button>
            <div id="status"></div>
        </div>
        <div class="right"></div>
        <div class="bottomleft"></div>
        <div class="bottom"></div>
        <div class="bottomright"></div>
        <div class="close"></div>
    </div>
`;

// Close button functionality
document.querySelector('.close').addEventListener('click', () => {
    window.go.main.App.Quit();
});

// Launch and embed Habbo
document.querySelector('#launchButton').addEventListener('click', async () => {
    const statusDiv = document.querySelector('#status');
    statusDiv.textContent = "Process started...";
    try {
        const result = await window.go.main.App.LaunchAndEmbedHabbo();
        statusDiv.textContent = result;
    } catch (error) {
        statusDiv.textContent = "Error: " + error;
    }
});

window.addEventListener('resize', () => {
    window.go.main.App.HandleResize()
        .catch(err => console.error('Error handling resize:', err));
});