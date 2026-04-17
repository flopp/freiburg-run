const on_load = function(f) {
    if (document.body === null) {
        document.addEventListener('DOMContentLoaded', () => {f()}, false);
    } else {
        f();
    }
}

const umami_track_event = function (name, data) {
    if (window.umami !== undefined) {
        window.umami.track(name, data);
    }
};

const parseGeo = function (s) {
    const match2coords = (m) => {
        const lat = parseFloat(m.groups.lat);
        const lng = parseFloat(m.groups.lng);
        return [lat, lng];
    };

    const re1 = /\s*N\s*(?<lat>\d+\.\d+)\s+E\s*(?<lng>\d+\.\d+)\s*$/gm;
    const match1 = re1.exec(s);
    if (match1 !== null) {
        return match2coords(match1);
    }

    const re2 = /\s*(?<lat>\d+\.\d+)\s*,\s*(?<lng>\d+\.\d+)\s*$/gm;
    const match2 = re2.exec(s);
    if (match2 !== null) {
        return match2coords(match2);
    }

    return null;
};

const onEach = (selector, callback) => {
    document.querySelectorAll(selector).forEach(callback);
};

const on = (selector, event, callback) => {
    document.querySelectorAll(selector).forEach(el => el.addEventListener(event, callback));
};

const decodeUnsignedIntegers = function (encoded) {
    const numbers = [];
    let index = 0;
    const len = encoded.length;
    while (index < len) {
        let num = 0;
        let shift = 0;
        while (true) {
            const b = encoded.charCodeAt(index++) - 63;
            num |= (b & 0x1f) << shift;
            if ((b & 0x20) === 0) break;
            shift += 5;
        }
        numbers.push(num);
    }
    return numbers;
};

const decodeSignedIntegers = function (encoded) {
    return decodeUnsignedIntegers(encoded).map(num => (num & 1) ? ~(num >> 1) : (num >> 1));
};

const decodeFloats = function (encoded) {
    const factor = 1e5;
    return decodeSignedIntegers(encoded).map(num => num / factor);
};

const decodeDeltas = function (encoded) {
    const dimension = 2;
    const lastNumbers = [];
    const numbers = decodeFloats(encoded);
    for (let i = 0, len = numbers.length; i < len;) {
        for (let d = 0; d < dimension; ++d, ++i) {
            numbers[i] = Math.round((lastNumbers[d] = numbers[i] + (lastNumbers[d] || 0)) * 100000) / 100000;
        }
    }
    return numbers;
};

const parsePolyline = function (encoded) {
    if (encoded === undefined || encoded.trim() === "") {
        return null;
    }
    const dimension = 2;
    const flatPoints = decodeDeltas(encoded);
    const points = [];
    for (let i = 0, len = flatPoints.length; i + (dimension -1) < len;) {
        const point = [];
        for (let dim = 0; dim < dimension; ++dim) {
            point.push(flatPoints[i++]);
        }
        points.push(point);
    }
    return points;
};

const loadMap = function (id) {
    const mapEl = document.getElementById(id);
    const cityName = mapEl.dataset.cityName || "Freiburg";
    const cityLat = parseFloat(mapEl.dataset.cityLat) || 47.996090;
    const cityLon = parseFloat(mapEl.dataset.cityLon) || 7.849400;

    const center = [cityLat, cityLon];
    const map = L.map(id, {gestureHandling: true}).setView(center, 15);

    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    L.circle(center, {
        color: '#3e8ed0',
        fill: false,
        weight: 1,
        radius: 25000
    }).addTo(map).bindPopup(`${cityName}, 25km`);
    L.circle(center, {
        color: '#3e8ed0',
        fill: false,
        weight: 1,
        radius: 50000
    }).addTo(map).bindPopup(`${cityName}, 50km`);

    const blueIcon = load_marker("");
    const greyIcon = load_marker("grey");
    const greenIcon = load_marker("green");
    const redIcon = load_marker("red");

    const markers = [];
    document.querySelectorAll(".event").forEach(el => {
        const geo = parseGeo(el.dataset.geo);
        if (geo !== null) {
            let icon = null;
            let zOffset = 0;
            switch (el.dataset.type) {
                case "Lauftreff":
                    zOffset = 1000;
                    icon = redIcon;
                    break;
                case "Lauf-Shop":
                    zOffset = 1000;
                    icon = greenIcon;
                    break;
                case "vergangene Veranstaltung":
                    zOffset = -1000;
                    icon = greyIcon;
                    break;
                case "Veranstaltung":
                default:
                    zOffset = 1000;
                    icon = blueIcon;
                    break;
            }

            const m = L.marker(geo, {icon: icon, zIndexOffset: zOffset});
            markers.push(m);
            m.addTo(map);
            if (el.dataset.time !== undefined) {
                m.bindPopup(`<a href="/${el.dataset.slug}">${el.dataset.name}</a><br>(${el.dataset.type})<br>${el.dataset.time}<br>${el.dataset.location}`);
            } else {
                m.bindPopup(`<a href="/${el.dataset.slug}">${el.dataset.name}</a><br>(${el.dataset.type})<br>${el.dataset.location}`);
            }
        }
    });

    const items = [{
        label: "Veranstaltung",
        type: "image",
        url: "/images/marker-icon.png",
    },{
        label: "vergangene Veranstaltung",
        type: "image",
        url: "/images/marker-grey-icon.png",
    },{
        label: "Lauftreff",
        type: "image",
        url: "/images/marker-red-icon.png",
    },{
        label: "Lauf-Shop",
        type: "image",
        url: "/images/marker-green-icon.png",
    }];
    items.push(
        {
            label: `25km um ${cityName}`,
            type: "image",
            url: "/images/circle-small.png"
        }, {
            label: `50km um ${cityName}`,
            type: "image",
            url: "/images/circle-big.png"
        }
    );
    const legend = L.control.Legend({
        title: "Legende",
        position: "bottomleft",
        collapsed: true,
        symbolWidth: 30,
        opacity: 1,
        column: 1,
        legends: items
    });
    legend.addTo(map);

    const group = new L.featureGroup(markers);
    map.fitBounds(group.getBounds(), {padding: L.point(40, 40)});
};

const loadParkrunMap = function (id, encodedTrack) {
    const map = L.map(id, {gestureHandling: true}).setView([48.000548, 7.804842], 15);

    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    const course = L.polyline(parsePolyline(encodedTrack));
    course.addTo(map);

    const blueIcon = load_marker("");
    const greyIcon = load_marker("grey");

    const parking = L.marker([48.000993,7.808887], {icon: greyIcon});
    parking.addTo(map);
    parking.bindPopup("Parkplatz");

    const tram = L.marker([47.999420,7.810088], {icon: greyIcon});
    tram.addTo(map);
    tram.bindPopup("Straßenbahn (Linie 3, Rohrgraben)");

    const meetingpoint = L.marker([48.001294,7.806489], {icon: blueIcon});
    meetingpoint.addTo(map);
    meetingpoint.bindPopup("Treffpunkt / Zielbereich");
};

const load_marker = function (color) {
    let url = "/images/marker-icon.png";
    let url2x = "/images/marker-icon-2x.png";
    if (color !== "") {
        url = "/images/marker-" + color + "-icon.png";
        url2x = "/images/marker-" + color + "-icon-2x.png";
    }
    const options = {
        iconAnchor: [12, 41],
        iconRetinaUrl: url2x,
        iconSize: [25, 41],
        iconUrl: url,
        popupAnchor: [1, -34],
        shadowSize: [41, 41],
        shadowUrl: "/images/marker-shadow.png",
        tooltipAnchor: [16, -28],
    };
    return L.icon(options);
}

const similarDistances = function(d1, d2, factor) {
    return ((d1 * (1.0-factor)) <= d2) && (d2 <= (d1 * (1.0+factor)));
}

const filter = (s, hiddenTags) => {
    let shown = 0;
    let hidden = 0;
    let hiddenTag = 0;
    const info = document.querySelector("#filter-info");
    const needle = s.toLowerCase().trim();

    // check if needle is a number (e.g. "10", "10.5", "10,5") and if so, use it as a distance filter
    let needleDistance = -1;
    if (needle !== "") {
        const re = /^(\d+[.,]?\d*)$/i;
        const match = needle.match(re);
        if (match) {
            needleDistance = parseFloat(match[1].replace(',', '.'));
        }
    }

    const items = new Array();
    document.querySelectorAll(".event, .event-separator").forEach(el => {
        const sep = el.previousSibling;
        if (sep === null) {
            items.push(null);
        }
        items.push(el);
    });

    let lastSep = null;
    items.forEach(el => {
        if (el === null) {
            lastSep = null;
        } else if (el.classList.contains("event-separator")) {
            if (lastSep !== null) {
                lastSep.classList.add("is-hidden");
            }
            lastSep = el;
        } else {
            // hide by tag
            if (hiddenTags.size != 0) {
                let found = false;
                el.querySelectorAll("[data-tag]").forEach(tagEl => {
                    if (tagEl.dataset.tag !== undefined) {
                        if (hiddenTags.has(tagEl.dataset.tag)) {
                            found = true;
                            return;
                        }
                    }
                });
                if (found) {
                    hiddenTag++;
                    el.classList.add("is-hidden");
                    return;
                }
            }

            // hide by search
            if (needle != "") {
                if (needleDistance > -1) {
                    let distances = [];
                    if (el.dataset.distancesParsed !== undefined) {
                        distances = JSON.parse(el.dataset.distancesParsed);
                    } else {
                        let distancesStr = el.dataset.distances;
                        // parse string into array "[5 10 42.3]" -> [5, 10, 42.3]
                        if (distancesStr !== undefined && distancesStr.trim() !== "") {
                            distancesStr = distancesStr.trim();
                            if (distancesStr.startsWith("[") && distancesStr.endsWith("]")) {
                                distancesStr = distancesStr.substring(1, distancesStr.length - 1);
                                distancesStr.split(" ").forEach(d => {
                                    const dist = parseFloat(d);
                                    if (!isNaN(dist)) {
                                        distances.push(dist);
                                    }
                                });
                            }
                        }
                        // store parsed distances in dataset for future use
                        el.dataset.distancesParsed = JSON.stringify(distances);
                    }
                    
                    let distanceMatch = false;
                    for (let d of distances) {
                        if (similarDistances(d, needleDistance, 0.1)) {
                            distanceMatch = true;
                            break;
                        }
                    }

                    if (!distanceMatch) {
                        hidden++;
                        el.classList.add("is-hidden");
                        return;
                    }
                } else {
                    const name = el.dataset.name.toLowerCase();
                    const location = el.dataset.location.toLowerCase();
                    if (!name.includes(needle) && !location.includes(needle)) {
                        hidden++;
                        el.classList.add("is-hidden");
                        return;
                    }
                }
            }
            
            // shown
            shown++;
            el.classList.remove("is-hidden");
            if (lastSep !== null) {
                lastSep.classList.remove("is-hidden");
            }
            lastSep = null;
        }
    });

    if (lastSep !== null) {
        lastSep.classList.add("is-hidden");
    }

    if (hidden != 0 || hiddenTag != 0) {
        let hiddenStr = ""
        if (hidden != 0) {
            hiddenStr = `, ${hidden} ${hidden!=1 ? "Einträge" : "Eintrag"} über Filter versteckt`;
        }
        let hiddenTagStr = ""
        if (hiddenTag != 0) {
            hiddenTagStr = `, ${hiddenTag} ${hiddenTag!=1 ? "Einträge" : "Eintrag"} über <a href="/tags.html">Kategorien</a> versteckt`;
        }
        let filterStr = "";
        if (needleDistance >= 0) {
            filterStr = `Filter nach Distanz: ${needleDistance} km ±10%; `;
        } else if (needle !== "") {
            // sanitize needle for HTML output (e.g. if it contains "<" or ">", escape it)
            const sanitizedNeedle = needle.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
            filterStr = `Filter nach Name/Ort: "${sanitizedNeedle}"; `;
        }
        info.innerHTML = `${filterStr}${shown} ${shown!=1 ? "Einträge" : "Eintrag"} angezeigt${hiddenStr}${hiddenTagStr}`;
        info.classList.remove("is-hidden");
    } else {
        info.innerHTML = `${shown} ${shown!=1 ? "Einträge" : "Eintrag"}`;
        info.classList.remove("is-hidden");
    }
};

function getLocalStorage() {
    let storage;
    try {
      storage = window["localStorage"];
      const x = "__storage_test__";
      storage.setItem(x, x);
      storage.removeItem(x);
      return storage;
    } catch (e) {
        return null;
    }
}

function createEl(tag, id, classes) {
    const el = document.createElement(tag);
    if (id !== undefined && id !== null && id !== "") {
        el.id = id;
    }
    if (classes !== undefined && classes !== null && classes !== "") {
        classes.split(" ").forEach(c => {
            el.classList.add(c);
        });
    }
    return el;
}

const WATCHLIST_STORAGE_KEY = "watchlist.v1";
const WATCHLIST_MAX_ITEMS = 100;

const todayDateKey = function() {
    const now = new Date();
    const yyyy = `${now.getFullYear()}`;
    const mm = `${now.getMonth() + 1}`.padStart(2, "0");
    const dd = `${now.getDate()}`.padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
};

const normalizeDateKey = function(s) {
    if (typeof s !== "string") {
        return "";
    }
    const cleaned = s.trim();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(cleaned)) {
        return "";
    }
    return cleaned;
};

const normalizeWatchlistItem = function(item) {
    if (item === null || typeof item !== "object") {
        return null;
    }

    const id = (item.id || "").toString().trim();
    const slug = (item.slug || "").toString().trim();
    if (id === "" || slug === "") {
        return null;
    }

    const addedAtNum = Number(item.addedAt);

    // Explicit category, or fall back to the slug prefix (e.g. "group/…" → "group")
    let category = (item.category || "").toString().trim();
    if (category === "") {
        const slashIdx = id.indexOf("/");
        if (slashIdx > 0) {
            category = id.substring(0, slashIdx);
        }
    }
    if (category !== "event" && category !== "group" && category !== "shop") {
        category = "event";
    }

    return {
        id: id,
        slug: slug,
        category: category,
        url: (item.url || "").toString().trim(),
        name: (item.name || "").toString().trim(),
        time: (item.time || "").toString().trim(),
        timeFrom: normalizeDateKey(item.timeFrom),
        timeTo: normalizeDateKey(item.timeTo),
        location: (item.location || "").toString().trim(),
        addedAt: Number.isFinite(addedAtNum) ? addedAtNum : Date.now(),
    };
};

const watchlistItemScore = function(item) {
    let score = 0;
    if (item.name !== "") score += 1;
    if (item.time !== "") score += 1;
    if (item.timeFrom !== "") score += 1;
    if (item.timeTo !== "") score += 1;
    if (item.location !== "") score += 1;
    if (item.url !== "") score += 1;
    return score;
};

const dedupeWatchlist = function(items) {
    const byId = new Map();
    items.forEach(item => {
        if (!byId.has(item.id)) {
            byId.set(item.id, item);
            return;
        }
        const existing = byId.get(item.id);
        if (watchlistItemScore(item) > watchlistItemScore(existing)) {
            byId.set(item.id, item);
        }
    });
    return Array.from(byId.values());
};

const sortWatchlist = function(items) {
    const sorted = [...items];
    sorted.sort((a, b) => {
        const aDate = a.timeFrom || a.timeTo;
        const bDate = b.timeFrom || b.timeTo;
        if (aDate === "" && bDate === "") {
            return a.name.localeCompare(b.name, "de", {sensitivity: "base"});
        }
        if (aDate === "") return 1;
        if (bDate === "") return -1;
        if (aDate < bDate) return -1;
        if (aDate > bDate) return 1;
        return a.name.localeCompare(b.name, "de", {sensitivity: "base"});
    });
    return sorted;
};

const prunePastWatchlistItems = function(items, todayKey) {
    return items.filter(item => {
        if (item.timeTo === "") {
            return true;
        }
        return item.timeTo >= todayKey;
    });
};

const getWatchlist = function(storage) {
    if (storage === null) {
        return [];
    }
    let raw;
    try {
        raw = storage.getItem(WATCHLIST_STORAGE_KEY);
    } catch (error) {
        return [];
    }
    if (raw === null || raw.trim() === "") {
        return [];
    }
    try {
        const parsed = JSON.parse(raw);
        if (!Array.isArray(parsed)) {
            return [];
        }
        const items = parsed
            .map(normalizeWatchlistItem)
            .filter(item => item !== null);
        return dedupeWatchlist(items);
    } catch (error) {
        return [];
    }
};

const saveWatchlist = function(storage, items) {
    if (storage === null) {
        return false;
    }
    try {
        storage.setItem(WATCHLIST_STORAGE_KEY, JSON.stringify(items));
        return true;
    } catch (error) {
        console.error("Saving watchlist failed", error);
        return false;
    }
};

const initWatchlist = function(storage) {
    const toggles = Array.from(document.querySelectorAll("[data-watchlist-toggle]"));
    const modal = document.getElementById("watchlist-modal");
    if (toggles.length === 0 && modal === null) {
        return;
    }

    const listEl = document.getElementById("watchlist-list");
    const emptyEl = document.getElementById("watchlist-empty");
    const warningEl = document.getElementById("watchlist-storage-warning");
    const todayKey = todayDateKey();

    if (storage === null) {
        toggles.forEach(toggle => {
            toggle.disabled = true;
            toggle.classList.add("is-disabled");
        });
        if (warningEl !== null) {
            warningEl.classList.remove("is-hidden");
        }
        return;
    }

    let watchlist = sortWatchlist(prunePastWatchlistItems(getWatchlist(storage), todayKey));
    saveWatchlist(storage, watchlist);

    const watchlistById = function() {
        const ids = new Set();
        watchlist.forEach(item => ids.add(item.id));
        return ids;
    };

    const syncToggleButtons = function() {
        const ids = watchlistById();
        toggles.forEach(toggle => {
            const id = (toggle.dataset.watchlistId || "").trim();
            const selected = id !== "" && ids.has(id);
            toggle.setAttribute("aria-pressed", selected ? "true" : "false");
            toggle.classList.toggle("is-watchlisted", selected);
        });
    };

    const makeWatchlistItem = function(item) {
        const li = createEl("li", null, "watchlist-item");
        const textWrap = createEl("div", null, "watchlist-item-text");

        const link = createEl("a", null, "watchlist-link");
        if (item.url !== "") {
            link.href = item.url;
        } else {
            link.href = `/${item.slug}`;
        }
        link.textContent = item.name !== "" ? item.name : item.slug;
        link.addEventListener("click", () => {
            umami_track_event("watchlist-open-item", {id: item.id, slug: item.slug});
        });
        textWrap.appendChild(link);

        const meta = [];
        if (item.time !== "") {
            meta.push(item.time);
        }
        if (item.location !== "") {
            meta.push(item.location);
        }
        if (meta.length > 0) {
            const metaEl = createEl("div", null, "watchlist-meta");
            metaEl.textContent = meta.join(" | ");
            textWrap.appendChild(metaEl);
        }

        const removeBtn = createEl("button", null, "button is-small is-light is-danger watchlist-remove");
        removeBtn.type = "button";
        removeBtn.textContent = "Entfernen";
        removeBtn.dataset.watchlistRemove = item.id;

        li.appendChild(textWrap);
        li.appendChild(removeBtn);
        return li;
    };

    const renderWatchlist = function() {
        if (listEl === null || emptyEl === null) {
            return;
        }

        listEl.innerHTML = "";
        if (watchlist.length === 0) {
            emptyEl.classList.remove("is-hidden");
            return;
        }
        emptyEl.classList.add("is-hidden");

        const sections = [
            {key: "event", label: "Veranstaltungen"},
            {key: "group", label: "Lauftreffs"},
            {key: "shop",  label: "Lauf-Shops"},
        ];
        const multipleCategories = sections.filter(s => watchlist.some(i => i.category === s.key)).length > 1;

        sections.forEach(({key, label}) => {
            const items = watchlist.filter(i => i.category === key);
            if (items.length === 0) {
                return;
            }
            if (multipleCategories) {
                const heading = createEl("p", null, "watchlist-category-heading");
                heading.textContent = label;
                listEl.appendChild(heading);
            }
            const ul = createEl("ul", null, "watchlist-items");
            items.forEach(item => ul.appendChild(makeWatchlistItem(item)));
            listEl.appendChild(ul);
        });
    };

    const refreshWatchlist = function(save) {
        watchlist = dedupeWatchlist(watchlist);
        watchlist = prunePastWatchlistItems(watchlist, todayDateKey());
        watchlist = sortWatchlist(watchlist);
        if (watchlist.length > WATCHLIST_MAX_ITEMS) {
            watchlist = watchlist.slice(0, WATCHLIST_MAX_ITEMS);
        }
        if (save) {
            saveWatchlist(storage, watchlist);
        }
        syncToggleButtons();
        renderWatchlist();

        document.querySelectorAll(".watchlist-count").forEach(el => {
            if (watchlist.length > 0) {
                el.textContent = watchlist.length;
                el.classList.remove("is-hidden");
            } else {
                el.classList.add("is-hidden");
            }
        });
    };

    const isElementVisible = function(el) {
        if (!(el instanceof HTMLElement)) {
            return false;
        }
        if (el.getClientRects().length === 0) {
            return false;
        }
        const style = window.getComputedStyle(el);
        return style.display !== "none" && style.visibility !== "hidden";
    };

    const findVisibleWatchlistTrigger = function(excludeEl) {
        const candidates = Array.from(document.querySelectorAll("[data-target='watchlist-modal']"));
        for (const candidate of candidates) {
            if (candidate === excludeEl) {
                continue;
            }
            if (isElementVisible(candidate)) {
                return candidate;
            }
        }
        return null;
    };

    const animateWatchlistAdd = function(sourceToggle) {
        if (!(sourceToggle instanceof HTMLElement)) {
            return;
        }
        if (window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
            return;
        }

        const target = findVisibleWatchlistTrigger(sourceToggle);
        if (!(target instanceof HTMLElement)) {
            return;
        }

        const startRect = sourceToggle.getBoundingClientRect();
        const targetRect = target.getBoundingClientRect();
        if (startRect.width === 0 || startRect.height === 0 || targetRect.width === 0 || targetRect.height === 0) {
            return;
        }

        const startX = startRect.left + startRect.width / 2;
        const startY = startRect.top + startRect.height / 2;
        const endX = targetRect.left + targetRect.width / 2;
        const endY = targetRect.top + targetRect.height / 2;

        const flyer = createEl("div", null, "watchlist-flyer");
        const star = document.createElement("span");
        star.classList.add("star-icon");
        flyer.appendChild(star);
        flyer.classList.add("watchlist-flyer");
        flyer.style.left = `${startX - 16}px`;
        flyer.style.top = `${startY - 16}px`;
        document.body.appendChild(flyer);

        const dx = endX - startX;
        const dy = endY - startY;
        const travel = flyer.animate([
            {transform: "translate(0, 0)", opacity: 0.95, offset: 0},
            {transform: `translate(${dx * 0.7}px, ${dy * 0.7}px)`, opacity: 0.9, offset: 0.65},
            {transform: `translate(${dx}px, ${dy}px)`, opacity: 0.1, offset: 1},
        ], {
            duration: 1000,
            easing: "cubic-bezier(0.22, 1, 0.36, 1)",
            fill: "forwards",
        });

        const cleanup = () => {
            flyer.remove();
        };
        travel.addEventListener("finish", cleanup, {once: true});
        travel.addEventListener("cancel", cleanup, {once: true});
    };

    toggles.forEach(toggle => {
        toggle.addEventListener("click", () => {
            const id = (toggle.dataset.watchlistId || "").trim();
            if (id === "") {
                return;
            }

            const existing = watchlist.some(item => item.id === id);
            if (existing) {
                watchlist = watchlist.filter(item => item.id !== id);
                umami_track_event("watchlist-remove", {id: id});
                refreshWatchlist(true);
                return;
            }

            watchlist.push(normalizeWatchlistItem({
                id: id,
                slug: (toggle.dataset.slug || "").trim(),
                category: (toggle.dataset.watchlistCategory || "").trim(),
                url: (toggle.dataset.url || "").trim(),
                name: (toggle.dataset.name || "").trim(),
                time: (toggle.dataset.time || "").trim(),
                timeFrom: (toggle.dataset.timeFrom || "").trim(),
                timeTo: (toggle.dataset.timeTo || "").trim(),
                location: (toggle.dataset.location || "").trim(),
                addedAt: Date.now(),
            }));
            watchlist = watchlist.filter(item => item !== null);
            umami_track_event("watchlist-add", {id: id});
            refreshWatchlist(true);

            animateWatchlistAdd(toggle);
        });
    });

    if (listEl !== null) {
        listEl.addEventListener("click", (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const id = target.dataset.watchlistRemove;
            if (id === undefined || id.trim() === "") {
                return;
            }

            watchlist = watchlist.filter(item => item.id !== id);
            umami_track_event("watchlist-remove", {id: id});
            refreshWatchlist(true);
        });
    }

    onEach('.modal-trigger', ($trigger) => {
        if ($trigger.dataset.target === "watchlist-modal") {
            $trigger.addEventListener('click', () => {
                umami_track_event('watchlist-open', {url: document.location.href});
            });
        }
    });

    refreshWatchlist(false);
};

const main = () => {
    // TAG FILTER, LOCAL STORAGE
    const storage = getLocalStorage();
    const hiddenTags = new Set();
    if (storage !== null) {
        let tags = storage.getItem("hiddenTags");
        if (tags !== null) {
            tags.split(",").forEach(tag => {
                tag = tag.trim();
                if (tag !== "") {
                    hiddenTags.add(tag);
                }
            });
        }
    }
    const tagTable = document.querySelector("#tag-table");
    if (tagTable !== null) {
        tagTable.querySelectorAll("[data-tag]").forEach(el => {
            if (storage !== null) {
                const tag = el.dataset.tag;
                el.checked = hiddenTags.has(tag);
                el.addEventListener('change', (event) => {
                    if (event.currentTarget.checked) {
                        hiddenTags.add(tag);
                    } else {
                        hiddenTags.delete(tag);
                    }
                    const tags = Array.from(hiddenTags).join(",");
                    storage.setItem("hiddenTags", tags);
                });
            } else {
                el.disabled = true;
            }
        });
    }

    // FILTER
    const filterInput = document.querySelector("#filter-input");
    if (filterInput !== null) {
        filterInput.addEventListener('input', (e) => {
            filter(e.target.value, hiddenTags);
        });
        document.querySelector("#filter-button-cancel").addEventListener('click', (e) => {
            filterInput.value = "";
            filter("", hiddenTags);
        });
        filter("", hiddenTags);
    }

    // WATCHLIST
    initWatchlist(storage);

    // SHARE BUTTONS
    onEach("[data-share]", shareButton => {
        const shareData = {
            title: shareButton.dataset.name,
            url: shareButton.dataset.url + "?utm_source=share_button",
        };

        if (navigator.canShare === undefined || !navigator.canShare(shareData)) {
            shareButton.classList.add("is-hidden");
            return;
        }

        shareButton.addEventListener('click', async (e) => {
            e.preventDefault();
            umami_track_event('share-click', {url: shareData.url});
            try {
                await navigator.share(shareData);
            } catch (error) {
                console.error("Error sharing:", error);
            }
        });
    });

    // CALENDARS
    onEach(".calendar-button", btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const modal = document.getElementById("calendar-modal");
            modal.querySelector(".event-name").innerText = btn.dataset.name;
            const googlecal = modal.querySelector(".calendar-google");
            googlecal.setAttribute("href", btn.dataset.googlecal);
            googlecal.setAttribute("data-umami-event", "calendar-click");
            const ics = modal.querySelector(".calendar-ics");
            ics.setAttribute("href", btn.dataset.calendar);
            ics.setAttribute("download", btn.dataset.calendarfile);
            ics.setAttribute("data-umami-event", "calendar-click");
            modal.classList.add("is-active");
            umami_track_event('calendar-click', {event: btn.dataset.name});
        });
    });

    // MAPS
    if (document.querySelector("#big-map") !== null) {
        loadMap("big-map");
    }

    const mapShowBtn = document.querySelector("#map-show-btn");
    const mapHideBtn = document.querySelector("#map-hide-btn");
    if (mapShowBtn !== null && mapHideBtn !== null) {
        mapShowBtn.addEventListener('click', () => {
            mapShowBtn.classList.add("is-hidden");
            mapHideBtn.classList.remove("is-hidden");
            const container = document.querySelector("#map-container");
            const mapDiv = createEl("div", "small-map", "");
            container.appendChild(mapDiv);
            if (container.dataset.type === "parkrun") {
                loadParkrunMap("small-map", container.dataset.track);
            } else {
                loadMap("small-map");
            }
        });
        mapHideBtn.addEventListener('click', () => {
            mapShowBtn.classList.remove("is-hidden");
            mapHideBtn.classList.add("is-hidden");
            document.querySelector("#small-map").remove();
        });

    }

    const eventMap = document.querySelector("#event-map");
    if (eventMap !== null) {
        const geo = parseGeo(eventMap.dataset.geo);
        const track = parsePolyline(eventMap.dataset.track);

        if (geo !== null) {
            const map = L.map('event-map', {gestureHandling: true}).setView(geo, 15);

            L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            }).addTo(map);

            const marker = L.marker(geo, {icon: load_marker("")});
            marker.addTo(map);
            marker.bindPopup(eventMap.dataset.name);
            if (track !== null) {
                const polyline = L.polyline(track, {color: '#3273dc'}).addTo(map);
                map.fitBounds(polyline.getBounds());
            }   
        }
    }

    // NAVBAR
    const burgersByTarget = new Map();
    const collectBurger = (burger, target) => {
        if (!burgersByTarget.has(target)) {
            burgersByTarget.set(target, []);
        }
        burgersByTarget.get(target).push(burger);
    }
    onEach('.navbar-burger', el => {
        collectBurger(el, el.dataset.target);
    });
    onEach('.navbar-burger-menu', el => {
        collectBurger(el, el.dataset.target);
    });
    burgersByTarget.forEach((burgers, target) => {
        burgers.forEach(el => {
            el.addEventListener('click', () => {
                if (!el.classList.contains('is-active')) {
                    umami_track_event('menu-open', {url: document.location.href});
                }
                burgers.forEach(be => {
                    be.classList.toggle('is-active');
                });
                document.getElementById(target).classList.toggle('is-active'); 
            });
        });
    });
    
    // MODALS
    function openModal($el) {
        $el.classList.add('is-active');
    }
    function closeModal($el) {
        $el.classList.remove('is-active');
    }
    function closeAllModals() {
        (document.querySelectorAll('.modal') || []).forEach(($modal) => {
            closeModal($modal);
        });
    }

    // Add a click event on buttons to open a specific modal
    onEach('.modal-trigger', ($trigger) => {
        const modal = $trigger.dataset.target;
        const $target = document.getElementById(modal);

        $trigger.addEventListener('click', (event) => {
            if ($trigger.tagName === "A") {
                event.preventDefault();
            }
            if ($target === null) {
                return;
            }
            openModal($target);
        });
    });

    // Add a click event on various child elements to close the parent modal
    onEach('.modal-background, .modal-close, .modal-card-head .delete, .modal-card-foot .button, .modal-card-body .close', ($close) => {
        const $target = $close.closest('.modal');

        $close.addEventListener('click', () => {
            closeModal($target);
        });
    });

    // Add a keyboard event to close all modals
    document.addEventListener('keydown', (e) => {
        if (e.key === "Escape") {
            closeAllModals();
        }
    });

    // DROPDOWNS
    const $clickableDropdowns = document.querySelectorAll(
        ".dropdown:not(.is-hoverable)",
    );

    if ($clickableDropdowns.length > 0) {
        $clickableDropdowns.forEach(($dropdown) => {
            const $button = $dropdown.querySelector("button");
            if (!$button) {
                return;
            }
            $button.addEventListener("click", (event) => {
                event.stopPropagation();
                $dropdown.classList.toggle("is-active");
            });
        });

        document.addEventListener("click", () => {
            closeDropdowns();
        });
    }

    function closeDropdowns() {
        $clickableDropdowns.forEach(($el) => {
            $el.classList.remove("is-active");
        });
    }

    // UMAMI
    onEach("a[target=_blank]", (a) => {
        if (a.getAttribute("data-umami-event") === null) {
            a.setAttribute('data-umami-event', 'outbound-link-click');
        }
        a.setAttribute('data-umami-event-url', a.href);
    });
    if (location.hash === '#disable-umami') {
        localStorage.setItem('umami.disabled', 'true');
        alert('Umami is now DISABLED in this browser.');
    }
    if (location.hash === '#enable-umami') {
        localStorage.removeItem('umami.disabled');
        alert('Umami is now ENABLED in this browser.');
    }

    // NOTIFICATIONS
    function isEmbedList() {
        return document.getElementById("embed-list") !== null;
    }
    function notificationGuard(id) {
        // don't show notifications if an element with id "embed-list" exists
        if (isEmbedList()) {
            console.log("Embed list detected, skipping notification.");
            return true;
        }

        // check if the notification has already been shown
        try {
            if (typeof localStorage !== "undefined") {
                const lastNotificationShown = localStorage.getItem("last-notification-shown");
                if (lastNotificationShown !== null) {
                    if (lastNotificationShown === id) {
                        console.log("Notification already shown, skipping.");
                        return true;
                    }
                }
                localStorage.setItem("last-notification-shown", id);
            }
        } catch (e) {
            console.error("LocalStorage not available, cannot store notification state.", e);
        }

        // if localStorage is not available, assume notification has not been shown
        return false;
    }

    function triggerNotificationOnce() {
        const notificationDataEl = document.getElementById("notification-data");
        if (notificationDataEl === null) {
            return;
        }

        let messages;
        try {
            messages = JSON.parse(notificationDataEl.getAttribute("data-messages"));
        } catch (e) {
            console.error("Failed to parse notification messages.", e);
            return;
        }

        if (!Array.isArray(messages) || messages.length === 0) {
            return;
        }

        // find the first message with an active start date (start <= today)
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        const activeMessage = messages.find(m => {
            if (m.start) {
                const start = new Date(m.start);
                if (!isNaN(start.getTime())) {
                    start.setHours(0, 0, 0, 0);
                    if (today < start) {
                        return false;
                    }
                }
            }
            return true;
        });

        if (activeMessage && !notificationGuard(`${activeMessage.id}`)) {
            setTimeout(() => {
                showNotification(activeMessage);
            }, 2000);
        }
    }

    function showNotification(notification) {
        if (!notification || !notification.content || !notification.class) {
            console.error("Invalid notification object.");
            return;
        }

        const existing = document.getElementById("notificationDiv");
        if (existing) {
            existing.remove();
        }

        const container = createEl("div", "notificationDiv", "container");
        document.body.appendChild(container);

        const div = createEl("div", null, "notification is-radiusless " + notification.class);
        container.appendChild(div);

        const closeButton = createEl("button", null, "delete");
        closeButton.onclick = () => container.remove();
        div.appendChild(closeButton);

        const contentDiv = createEl("div");
        contentDiv.innerHTML = notification.content;
        div.appendChild(contentDiv);
        
        setTimeout(() => {
            container.classList.add("active");
        }, 10);
    }

    window.addEventListener("DOMContentLoaded", triggerNotificationOnce);
};

on_load(main);