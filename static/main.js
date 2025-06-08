var on_load = function(f) {
    if (document.body === null) {
        document.addEventListener('DOMContentLoaded', () => {f()}, false);
    } else {
        f();
    }
}

var toggle_menuitem = function (id) {
    var next = document.getElementById(id);
    var current = document.querySelector(".navbar-item.is-active");
    if (next != null && next !== current) {
        if (current !== null) {
            current.classList.remove("is-active");
        }
        next.classList.add("is-active");
    }
};

const parseGeo = function (s) {
    const re1 = /\s*N\s*(?<lat>\d+\.\d+)\s+E\s*(?<lng>\d+\.\d+)\s*$/gm;
    const match1 = re1.exec(s);
    if (match1 !== null) {
        let lat = parseFloat(match1.groups.lat);
        let lng = parseFloat(match1.groups.lng);
        return [lat, lng];
    }

    const re2 = /\s*(?<lat>\d+\.\d+)\s*,\s*(?<lng>\d+\.\d+)\s*$/gm;
    const match2 = re2.exec(s);
    if (match2 !== null) {
        let lat = parseFloat(match2.groups.lat);
        let lng = parseFloat(match2.groups.lng);
        return [lat, lng];
    }

    return null;
};

const loadMap = function (id) {
    var map = L.map(id, {gestureHandling: true}).setView([48.000548, 7.804842], 15);

    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    var freiburg = [47.996090, 7.849400];
    L.circle(freiburg, {
        color: '#3e8ed0',
        fill: false,
        weight: 1,
        radius: 25000
    }).addTo(map).bindPopup("Freiburg, 25km");
    L.circle(freiburg, {
        color: '#3e8ed0',
        fill: false,
        weight: 1,
        radius: 50000
    }).addTo(map).bindPopup("Freiburg, 50km")

    let blueIcon = load_marker("");
    let greyIcon = load_marker("grey");
    let greenIcon = load_marker("green");
    let redIcon = load_marker("red");

    let markers = [];
    document.querySelectorAll(".event").forEach(el => {
        let geo = parseGeo(el.dataset.geo);
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

            let m = L.marker(geo, {icon: icon, zIndexOffset: zOffset});
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
            label: "25km um Freiburg",
            type: "image",
            url: "/images/circle-small.png"
        }, {
            label: "50km um Freiburg",
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

    var group = new L.featureGroup(markers);
    map.fitBounds(group.getBounds(), {padding: L.point(40, 40)});
};

const loadParkrunMap = function (id) {
    var map = L.map(id, {gestureHandling: true}).setView([48.000548, 7.804842], 15);

    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    var course = L.polyline(parkrunTrack);
    course.addTo(map);

    let blueIcon = load_marker("");
    let greyIcon = load_marker("grey");

    let parking = L.marker([48.000993,7.808887], {icon: greyIcon});
    parking.addTo(map);
    parking.bindPopup("Parkplatz");

    let tram = L.marker([47.999420,7.810088], {icon: greyIcon});
    tram.addTo(map);
    tram.bindPopup("Straßenbahn (Linie 3, Rohrgraben)");

    let meetingpoint = L.marker([48.001294,7.806489], {icon: blueIcon});
    meetingpoint.addTo(map);
    meetingpoint.bindPopup("Treffpunkt / Zielbereich");
};

var load_marker = function (color) {
    let url = "/images/marker-icon.png";
    let url2x = "/images/marker-icon-2x.png";
    if (color !== "") {
        url = "/images/marker-" + color + "-icon.png";
        url2x = "/images/marker-" + color + "-icon-2x.png";
    }
    let options = {
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

var filter = (s, hiddenTags) => {
    let shown = 0;
    let hidden = 0;
    let hiddenTag = 0;
    let info = document.querySelector("#filter-info");
    let needle = s.toLowerCase().trim();

    let items = new Array();
    document.querySelectorAll(".event, .event-separator").forEach(el => {
        var sep = el.previousSibling;
        if (sep === null) {
            items.push(null);
        }
        items.push(el);
    });

    lastSep = null;
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
                var found = false;
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
                let name = el.dataset.name.toLowerCase();
                let location = el.dataset.location.toLowerCase();
                if (!name.includes(needle) && !location.includes(needle)) {
                    hidden++;
                    el.classList.add("is-hidden");
                    return;
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
        var hiddenStr = ""
        if (hidden != 0) {
            hiddenStr = `, ${hidden} ${hidden!=1 ? "Einträge" : "Eintrag"} über Filter versteckt`;
        }
        var hiddenTagStr = ""
        if (hiddenTag != 0) {
            hiddenTagStr = `, ${hiddenTag} ${hiddenTag!=1 ? "Einträge" : "Eintrag"} über <a href="/tags.html">Kategorien</a> versteckt`;
        }
        info.innerHTML = `${shown} ${shown!=1 ? "Einträge" : "Eintrag"} angezeigt${hiddenStr}${hiddenTagStr}`;
        info.classList.remove("is-hidden");
    } else {
        info.classList.add("is-hidden");
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

function createEl(tag, classes) {
    const el = document.createElement(tag);
    if (classes !== undefined) {
        classes.split(" ").forEach(c => {
            el.classList.add(c);
        });
    }
    return el;
} 

var main = () => {
    // TAG FILTER, LOCAL STORAGE
    var storage = getLocalStorage();
    var hiddenTags = new Set();
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
    var tagTable = document.querySelector("#tag-table");
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
                    var tags = Array.from(hiddenTags).join(",");
                    storage.setItem("hiddenTags", tags);
                });
            } else {
                el.disabled = true;
            }
        });
    }

    // FILTER
    var filterInput = document.querySelector("#filter-input");
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

    // CALENDARS
    document.querySelectorAll(".calendar-button").forEach(dropdown => {
        dropdown.classList.add("dropdown");

        const dropdownTrigger = createEl("div", "dropdown-trigger");
        const dropdownTriggerButton = createEl("button", "button is-text is-small py-1 ml-1");
        dropdownTriggerButton.innerHTML = "Zum Kalender hinzufügen";
        dropdownTrigger.appendChild(dropdownTriggerButton);
        dropdown.appendChild(dropdownTrigger);

        const dropdownMenu = createEl("div", "dropdown-menu");
        const dropdownContent = createEl("div", "dropdown-content");

        const hint = createEl("p", "dropdown-item is-italic");
        hint.innerHTML = "Da genaue Start- & End-Zeiten unbekannt sind, werden Events als Ganztages-Einträge angelegt.";
        dropdownContent.appendChild(hint);

        const div1 = createEl("hr", "dropdown-divider");
        dropdownContent.appendChild(div1);

        const googlecal = createEl("a", "dropdown-item");
        googlecal.setAttribute("href", dropdown.dataset.googlecal);
        googlecal.setAttribute("data-umami-event", "calendar-click");
        googlecal.setAttribute("rel", "nofollow");
        googlecal.setAttribute("target", "_blank");
        googlecal.innerHTML = "Google Calendar";
        dropdownContent.appendChild(googlecal);

        const div2 = createEl("hr", "dropdown-divider");
        dropdownContent.appendChild(div2);

        const ics = createEl("a", "dropdown-item");
        ics.setAttribute("href", dropdown.dataset.calendar);
        ics.setAttribute("data-umami-event", "calendar-click");
        ics.setAttribute("rel", "nofollow");
        ics.setAttribute("target", "_blank");
        ics.innerHTML = "Outlook, Apple Calendar & andere (.ics)";
        dropdownContent.appendChild(ics);

        dropdownMenu.appendChild(dropdownContent);
        dropdown.appendChild(dropdownMenu);
    });

    // MAPS
    var bigMapId = "";
    if (document.querySelector("#big-map") !== null) {
        bigMapId = "big-map";
    } else if (document.querySelector("#serie-map") !== null) {
        bigMapId = "serie-map";
    }
    if (bigMapId !== "") {
        loadMap(bigMapId);
    }

    const mapShowBtn = document.querySelector("#map-show-btn");
    const mapHideBtn = document.querySelector("#map-hide-btn");
    if (mapShowBtn !== null && mapHideBtn !== null) {
        mapShowBtn.addEventListener('click', () => {
            mapShowBtn.classList.add("is-hidden");
            mapHideBtn.classList.remove("is-hidden");
            const container = document.querySelector("#map-container");
            const mapDiv = document.createElement("div");
            mapDiv.id = "small-map";
            container.appendChild(mapDiv);
            if (container.dataset.type === "parkrun") {
                loadParkrunMap("small-map");
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

    let eventMap = document.querySelector("#event-map");
    if (eventMap !== null) {
        let geo = parseGeo(eventMap.dataset.geo);
        if (geo !== null) {
            var map = L.map('event-map', {gestureHandling: true}).setView(geo, 15);

            L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            }).addTo(map);

            let marker = L.marker(geo, {icon: load_marker("")});
            marker.addTo(map);
            marker.bindPopup(eventMap.dataset.name);
        }
    }

    // NAVBAR
    document.querySelectorAll('.navbar-burger').forEach(el => {
        el.addEventListener('click', () => {
            const target = el.dataset.target;
            el.classList.toggle('is-active');
            document.getElementById(target).classList.toggle('is-active'); 
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
    (document.querySelectorAll('.modal-trigger') || []).forEach(($trigger) => {
        const modal = $trigger.dataset.target;
        const $target = document.getElementById(modal);

        $trigger.addEventListener('click', () => {
            openModal($target);
        });
    });

    // Add a click event on various child elements to close the parent modal
    (document.querySelectorAll('.modal-background, .modal-close, .modal-card-head .delete, .modal-card-foot .button, .modal-card-body .close') || []).forEach(($close) => {
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
    document.querySelectorAll("a[target=_blank]").forEach((a) => {
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
    function notificationGuard(id) {
        const lastNotificationShown = localStorage.getItem("last-notification-shown");
        if (lastNotificationShown !== null) {
            const lastNotificationId = parseInt(lastNotificationShown, 10);
            if (lastNotificationId === id) {
                console.log("Notification already shown, skipping.");
                return true;
            }
        }
        localStorage.setItem("last-notification-shown", `${id}`);
        return false;
    }

    function triggerNotificationOnce(force=false) {
        const notification = {
            id: 1,
            content: "<b>Gewinnspiel zum \"Tote Mann Berglauf 2025\"</b><br>Irgendwo auf <b>freiburg.run</b> sind Instruktionen für das Gewinnspiel versteckt - finde sie bis zum 31.06.2025 und nimm an der Verlosung eines Startplatzes für den <b>Tote Mann Berglauf 2025</b> teil!",
            class: "is-warning",
        };

        if (force || !notificationGuard(notification.id)) {
            setTimeout(() => {
                showNotification(notification);
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

        const container = document.createElement("div");
        container.id = "notificationDiv";
        container.className = "container";
        container.style.position = "fixed";
        container.style.zIndex = "2000";
        container.style.left = "50%";
        container.style.bottom = "0px";
        container.style.transform = "translate(-50%, 100%)";
        container.style.transition = "transform 1s cubic-bezier(.4,0,.2,1)";
        document.body.appendChild(container);

        const div = document.createElement("div");
        div.className = "notification is-radiusless " + notification.class;
        container.appendChild(div);

        const closeButton = document.createElement("button");
        closeButton.className = "delete";
        closeButton.onclick = () => container.remove();
        div.appendChild(closeButton);

        const contentDiv = document.createElement("div");
        contentDiv.innerHTML = notification.content;
        div.appendChild(contentDiv);
        
        setTimeout(() => {
            container.style.transform = "translate(-50%, 0)";
        }, 10);
    }

    window.addEventListener("DOMContentLoaded", triggerNotificationOnce);
};

on_load(main);