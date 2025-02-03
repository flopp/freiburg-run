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

    let cafe = L.marker([47.997826,7.807831], {icon: greyIcon});
    cafe.addTo(map);
    cafe.bindPopup("Lio's Café");

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

var filter = (s) => {
    let shown = 0;
    let hidden = 0;
    let info = document.querySelector("#filter-info");
    if (s == "") {
        document.querySelectorAll(".event").forEach(el => {
            shown++;
            el.classList.remove("is-hidden");
        });
        info.classList.add("is-hidden");
    } else {
        let needle = s.toLowerCase().trim();
        document.querySelectorAll(".event").forEach(el => {
            let name = el.dataset.name.toLowerCase();
            if (name.includes(needle)) {
                shown++;
                el.classList.remove("is-hidden");
            } else {
                hidden++;
                el.classList.add("is-hidden");
            }
        });
        info.innerHTML = `${shown} ${shown!=1 ? "Einträge" : "Eintrag"} angezeigt, ${hidden} ${hidden!=1 ? "Einträge" : "Eintrag"} versteckt`;
        info.classList.remove("is-hidden");
    }
};

var main = () => {
    document.querySelectorAll('.navbar-burger').forEach(el => {
        el.addEventListener('click', () => {
            const target = el.dataset.target;
            el.classList.toggle('is-active');
            document.getElementById(target).classList.toggle('is-active'); 
        });
    });
    
    var filterInput = document.querySelector("#filter-input");
    if (filterInput !== null) {
        filterInput.addEventListener('input', (e) => {
            filter(e.target.value);
        });
        document.querySelector("#filter-button-cancel").addEventListener('click', (e) => {
            filterInput.value = "";
            filter("");
        });
    }

    var bigMapId = "";
    if (document.querySelector("#big-map") !== null) {
        bigMapId = "big-map";
    } else if (document.querySelector("#serie-map") !== null) {
        bigMapId = "serie-map";
    }
    if (bigMapId !== "") {
        loadMap(bigMapId);
    }

    const mapBtn = document.querySelector("#map-toggle-btn");
    if (mapBtn !== null) {
        mapBtn.addEventListener('click', () => {
            mapBtn.remove();
            const mapDiv = document.querySelector("#map-toggle");
            mapDiv.classList.add("is-active"); 
            loadMap("map-toggle");
        });
    }

    const parkrunMapBtn = document.querySelector("#parkrun-map-toggle-btn");
    if (parkrunMapBtn !== null) {
        parkrunMapBtn.addEventListener('click', () => {
            parkrunMapBtn.remove();
            const mapDiv = document.querySelector("#map-toggle");
            mapDiv.classList.add("is-active"); 
            loadParkrunMap("map-toggle");
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

    // Functions to open and close a modal
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
            if (window.goatcounter && window.goatcounter.count) {
                window.goatcounter.count({
                    path:  $trigger.dataset.target,
                    title: $trigger.dataset.target,
                    referrer: window.location.href || '',
                    event: true,
                });
            }
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
};

on_load(main);