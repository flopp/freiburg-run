function enableSnow() {
    // SNOW
    const snowCount = 200;
    const snowWrapper = document.getElementById("snow");
    let bodyHeightPx = null;
    let pageHeightVh = null;

    function setHeightVariables() {
        bodyHeightPx = document.body.offsetHeight;
        pageHeightVh = (100 * bodyHeightPx / window.innerHeight);
    }
    function generateSnow(snowDensity = 200) {
        snowDensity -= 1;
        snowWrapper.innerHTML = '';
        for (let i = 0; i < snowDensity; i++) {
            let board = document.createElement('div');
            board.className = "snowflake";
            snowWrapper.appendChild(board);
        }
    }
    function getOrCreateCSSElement() {
        let cssElement = document.getElementById("psjs-css");
        if (cssElement) return cssElement;

        cssElement = document.createElement('style');
        cssElement.id = 'psjs-css';
        document.head.appendChild(cssElement);
        return cssElement;
    }

    // Append style for each snowflake to the head
    function addCSS(rule) {
        const cssElement = getOrCreateCSSElement();
        cssElement.innerHTML = rule; // safe to use innerHTML
        document.head.appendChild(cssElement);
    }

    // Math
    function randomInt(value = 100) {
        return Math.floor(Math.random() * value) + 1;
    }

    function randomIntRange(min, max) {
        min = Math.ceil(min);
        max = Math.floor(max);
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }

    function getRandomArbitrary(min, max) {
        return Math.random() * (max - min) + min;
    }
    function generateSnowCSS(snowDensity = 200) {
        let rule = "";
        for (let i = 1; i < snowDensity; i++) {
            let randomX = Math.random() * 100; // vw
            let randomOffset = Math.random() * 10 // vw;
            let randomXEnd = randomX + randomOffset;
            let randomXEndYoyo = randomX + (randomOffset / 2);
            let randomYoyoTime = getRandomArbitrary(0.3, 0.8);
            let randomYoyoY = randomYoyoTime * pageHeightVh; // vh
            let randomScale = getRandomArbitrary(0.2, 1.0);
            let fallDuration = randomIntRange(10, pageHeightVh / 10 * 3); // s
            let fallDelay = randomInt(pageHeightVh / 10 * 3) * -1; // s
            let opacity = getRandomArbitrary(0.4, 1.0);

            rule += `
            .snowflake:nth-child(${i}) {
                opacity: ${opacity};
                transform: translate(${randomX}vw, -10px) scale(${randomScale});
                animation: fall-${i} ${fallDuration}s ${fallDelay}s linear infinite;
            }
            @keyframes fall-${i} {
                ${randomYoyoTime * 100}% {
                transform: translate(${randomXEnd}vw, ${randomYoyoY}vh) scale(${randomScale});
                }
                to {
                transform: translate(${randomXEndYoyo}vw, ${pageHeightVh}vh) scale(${randomScale});
                }
            }
            `
        }
        addCSS(rule);
    }

    function createSnow() {
        setHeightVariables();
        generateSnowCSS(snowCount);
        generateSnow(snowCount);
    };

    if (snowWrapper !== null && !isEmbedList()) {
        window.addEventListener('resize', createSnow);
        createSnow();
    }
}