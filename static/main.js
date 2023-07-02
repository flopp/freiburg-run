var on_load = function(f) {
    if (document.body === null) {
        document.addEventListener('DOMContentLoaded', () => {f()}, false);
    } else {
        f();
    }
}

const parseAllDates = function (s) {
    const dates = new Array();
    const re = /(?<day>\d\d)\.(?<month>\d\d)\.(?<year>\d\d\d\d)/g;
    const matches = [...s.matchAll(re)];
    matches.forEach(m => {
        dates.push(new Date(
            parseInt(m.groups.year),
            parseInt(m.groups.month)-1,
            parseInt(m.groups.day),
            0, 0, 0));
    });
    return dates;
};

const parseDate = function (s) {
    const re1 = /\s*(?<day>\d\d?)\.(?<month>\d\d?)\.(?<year>\d\d\d\d)\s*$/gm;
    const match1 = re1.exec(s);
    if (match1 !== null) {
        return new Date(
            parseInt(match1.groups.year),
            parseInt(match1.groups.month)-1,
            parseInt(match1.groups.day),
            0, 0, 0);
    }

    const re2 = /\s*(?<year>\d\d\d\d)-(?<month>\d\d)-(?<day>\d\d)\s*$/gm;
    const match2 = re2.exec(s);
    if (match2 !== null) {
        return new Date(
            parseInt(match2.groups.year),
            parseInt(match2.groups.month)-1,
            parseInt(match2.groups.day),
            0, 0, 0);
    }

    return NaN;
};

var update_events = function (show_past) {
    if (show_past) {
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "1") {
                return;
            }
    
            el.classList.remove("is-hidden");
        });
        return;
    }

    const now = new Date();
    const nowY = now.getFullYear();
    const nowM = now.getMonth() + 1;
    const nowD = now.getDate() - 2;



    document.querySelectorAll(".event").forEach(el => {
        if (el.dataset.pending === "1") {
            return;
        }

        const dateEl = el.children[0];
        const dateString = dateEl.textContent;
        let dateFound = false;
        let someAfter = false;
        /*
        console.log(dateString);
        parseAllDates(dateString).forEach(date => {
            dateFound = true;
            const day = 24 * 60 * 60 * 1000;
            if (date + day > now) {
                someAfter = true;
            }
        });
        */

        const dateRegex = /(\d\d)\.(\d\d)\.(\d\d\d\d)/g;
        const matches = [...dateString.matchAll(dateRegex)];
        matches.forEach(date => {
            dateFound = true;

            const y = parseInt(date[3]);
            if (y > nowY) {
                someAfter = true;
                return;
            }
            if (y < nowY) {
                return;
            }

            const m = parseInt(date[2]);
            if (m > nowM) {
                someAfter = true;
                return;
            }
            if (m < nowM) {
                return;
            }

            const d = parseInt(date[1]);
            if (d > nowD) {
                someAfter = true;
                return;
            }
            if (y == nowD) {
                someAfter = true;
            }
        });

        if (dateFound) {
            if (someAfter) {
                el.classList.remove("is-hidden");
            } else {
                el.classList.add("is-hidden");
            }
        } else {
            el.classList.remove("is-hidden");
        }
    });
}

var toggle_map = function (mapDiv, leafletMap, show) {
    if (show) {
        mapDiv.classList.remove("is-hidden");
        leafletMap.invalidateSize();
    } else {
        mapDiv.classList.add("is-hidden");
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

    var mapDiv = document.querySelector("#map");
    var leafletMap = null;
    if (mapDiv !== null) {
        var map = L.map('map').setView([51.505, -0.09], 13);
        leafletMap = map;

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

        let blueOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let blueIcon = L.icon(blueOptions);

        let greyOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-grey-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-grey-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let greyIcon = L.icon(greyOptions);

        let hasPending = false;
        let markers = [];
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "0") {
                return;
            }
            hasPending = true;
            let geo = parseGeo(el.dataset.geo);
            if (geo !== null) {
                let m = L.marker(geo, {icon: greyIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}<br>NICHT BESTÄTIGT`);
            }
        });
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "1") {
                return;
            }
            let geo = parseGeo(el.dataset.geo);
            if (geo !== null) {
                let m = L.marker(geo, {icon: blueIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}`);
            }
            let added = parseDate(el.dataset.added);
            const maxAge = 7 * 86400 * 1000; /* 7 days */
            if (added !== NaN && (Date.now() - added) < maxAge) {
                const dateEl = el.children[0];
                dateEl.classList.add("is-success");
                dateEl.textContent += " (neu)";
            }
        });

        const itemName = document.querySelector("body").dataset.itemtype;
        const items = [{
            label: itemName,
            type: "image",
            url: "images/marker-icon.png",
        }];
        if (hasPending) {
            items.push({
                label: `${itemName} (unbestätigt)`,
                type: "image",
                url: "images/marker-grey-icon.png"
            });
        }
        items.push(
            {
                label: "25km um Freiburg",
                type: "image",
                url: "images/circle-small.png"
            }, {
                label: "50km um Freiburg",
                type: "image",
                url: "images/circle-big.png"
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
    }

    if (document.querySelector("#parkrun-map") !== null) {
        mapDiv = document.querySelector("#parkrun-map-wrapper");
        var map = L.map('parkrun-map').setView([48.000548, 7.804842], 15);
        leafletMap = map;

        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }).addTo(map);

        var course = L.polyline(parkrunTrack);
        course.addTo(map);

        let blueOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let blueIcon = L.icon(blueOptions);

        let greyOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-grey-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-grey-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let greyIcon = L.icon(greyOptions);
        
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
    }

    var checkbox = document.querySelector("#show-past-events");
    if (checkbox !== null) {
        checkbox.addEventListener('change', (event) => {
            update_events(checkbox.checked);
        });
        update_events(checkbox.checked);
    }

    if (mapDiv !== null) {
        var checkboxMap = document.querySelector("#show-map");
        if (checkboxMap !== null) {
            checkboxMap.addEventListener('change', (event) => {
                toggle_map(mapDiv, leafletMap, checkboxMap.checked);
            });
            toggle_map(mapDiv, leafletMap, checkboxMap.checked);
        }
    }
};

on_load(main);


(function() {
	'use strict';

	if (window.counter && window.counter.vars)  // Compatibility with very old version; do not use.
		window.counter = window.counter.vars
	else
		window.counter = window.counter || {}

	// Load settings from data-counter-settings.
	var s = document.querySelector('script[data-counter]')
	if (s && s.dataset.counterSettings) {
		try         { var set = JSON.parse(s.dataset.counterSettings) }
		catch (err) { console.error('invalid JSON in data-counter-settings: ' + err) }
		for (var k in set)
			if (['no_onload', 'no_events', 'allow_local', 'allow_frame', 'path', 'title', 'referrer', 'event'].indexOf(k) > -1)
				window.counter[k] = set[k]
	}

	var enc = encodeURIComponent

	// Get all data we're going to send off to the counter endpoint.
	var get_data = function(vars) {
		var data = {
			p: (vars.path     === undefined ? counter.path     : vars.path),
			r: (vars.referrer === undefined ? counter.referrer : vars.referrer),
			t: (vars.title    === undefined ? counter.title    : vars.title),
			e: !!(vars.event || counter.event),
			s: [window.screen.width, window.screen.height, (window.devicePixelRatio || 1)],
			b: is_bot(),
			q: location.search,
		}

		var rcb, pcb, tcb  // Save callbacks to apply later.
		if (typeof(data.r) === 'function') rcb = data.r
		if (typeof(data.t) === 'function') tcb = data.t
		if (typeof(data.p) === 'function') pcb = data.p

		if (is_empty(data.r)) data.r = document.referrer
		if (is_empty(data.t)) data.t = document.title
		if (is_empty(data.p)) data.p = get_path()

		if (rcb) data.r = rcb(data.r)
		if (tcb) data.t = tcb(data.t)
		if (pcb) data.p = pcb(data.p)
		return data
	}

	// Check if a value is "empty" for the purpose of get_data().
	var is_empty = function(v) { return v === null || v === undefined || typeof(v) === 'function' }

	// See if this looks like a bot; there is some additional filtering on the
	// backend, but these properties can't be fetched from there.
	var is_bot = function() {
		// Headless browsers are probably a bot.
		var w = window, d = document
		if (w.callPhantom || w._phantom || w.phantom)
			return 150
		if (w.__nightmare)
			return 151
		if (d.__selenium_unwrapped || d.__webdriver_evaluate || d.__driver_evaluate)
			return 152
		if (navigator.webdriver)
			return 153
		return 0
	}

	// Object to urlencoded string, starting with a ?.
	var urlencode = function(obj) {
		var p = []
		for (var k in obj)
			if (obj[k] !== '' && obj[k] !== null && obj[k] !== undefined && obj[k] !== false)
				p.push(enc(k) + '=' + enc(obj[k]))
		return '?' + p.join('&')
	}

	// Show a warning in the console.
	var warn = function(msg) {
		if (console && 'warn' in console)
			console.warn('counter: ' + msg)
	}

	// Get the endpoint to send requests to.
	var get_endpoint = function() {
		return "https://s.freiburg.run/i";
	}

	// Get current path.
	var get_path = function() {
		var loc = location,
			c = document.querySelector('link[rel="canonical"][href]')
		if (c) {  // May be relative or point to different domain.
			var a = document.createElement('a')
			a.href = c.href
			if (a.hostname.replace(/^www\./, '') === location.hostname.replace(/^www\./, ''))
				loc = a
		}
		return (loc.pathname + loc.search) || '/'
	}

	// Filter some requests that we (probably) don't want to count.
	counter.filter = function() {
		if ('visibilityState' in document && document.visibilityState === 'prerender')
			return 'visibilityState'
		if (!counter.allow_frame && location !== parent.location)
			return 'frame'
		if (!counter.allow_local && location.hostname.match(/(localhost$|^127\.|^10\.|^172\.(1[6-9]|2[0-9]|3[0-1])\.|^192\.168\.|^0\.0\.0\.0$)/))
			return 'localhost'
		if (!counter.allow_local && location.protocol === 'file:')
			return 'localfile'
		if (localStorage && localStorage.getItem('skipgc') === 't')
			return 'disabled with #toggle-counter'
		return false
	}

	// Get URL to send to counter.
	window.counter.url = function(vars) {
		var data = get_data(vars || {})
		if (data.p === null)  // null from user callback.
			return
		data.rnd = Math.random().toString(36).substr(2, 5)  // Browsers don't always listen to Cache-Control.

		var endpoint = get_endpoint()
		if (!endpoint)
			return warn('no endpoint found')

		return endpoint + urlencode(data)
	}

	// Count a hit.
	window.counter.count = function(vars) {
		var f = counter.filter()
		if (f)
			return warn('not counting because of: ' + f)

		var url = counter.url(vars)
		if (!url)
			return warn('not counting because path callback returned null')

		var img = document.createElement('img')
		img.src = url
		img.style.position = 'absolute'  // Affect layout less.
		img.style.bottom = '0px'
		img.style.width = '1px'
		img.style.height = '1px'
		img.loading = 'eager'
		img.setAttribute('alt', '')
		img.setAttribute('aria-hidden', 'true')

		var rm = function() { if (img && img.parentNode) img.parentNode.removeChild(img) }
		img.addEventListener('load', rm, false)
		document.body.appendChild(img)
	}

	// Get a query parameter.
	window.counter.get_query = function(name) {
		var s = location.search.substr(1).split('&')
		for (var i = 0; i < s.length; i++)
			if (s[i].toLowerCase().indexOf(name.toLowerCase() + '=') === 0)
				return s[i].substr(name.length + 1)
	}

	// Track click events.
	window.counter.bind_events = function() {
		if (!document.querySelectorAll)  // Just in case someone uses an ancient browser.
			return

		var send = function(elem) {
			return function() {
				counter.count({
					event:    true,
					path:     (elem.href || elem.dataset.counterClick || elem.name || elem.id || ''),
					title:    (elem.dataset.counterTitle || elem.title || (elem.innerHTML || '').substr(0, 200) || ''),
					referrer: (elem.dataset.counterReferrer || elem.dataset.counterReferral || ''),
				})
			}
		}

		Array.prototype.slice.call(document.querySelectorAll("a")).forEach(function(elem) {
			if (!(elem.target === "_blank")) {
                return
            }
            if (elem.dataset.counterBound)
				return
			var f = send(elem)
			elem.addEventListener('click', f, false)
			elem.addEventListener('auxclick', f, false)  // Middle click.
			elem.dataset.counterBound = 'true'
		})
	}

	// Make it easy to skip your own views.
	if (location.hash === '#toggle-counter') {
		if (localStorage.getItem('skipgc') === 't') {
			localStorage.removeItem('skipgc', 't')
			alert('counter tracking is now ENABLED in this browser.')
		}
		else {
			localStorage.setItem('skipgc', 't')
			alert('counter tracking is now DISABLED in this browser until ' + location + ' is loaded again.')
		}
	}

	on_load(function() {
			// 1. Page is visible, count request.
			// 2. Page is not yet visible; wait until it switches to 'visible' and count.
			// See #487
			if (!('visibilityState' in document) || document.visibilityState === 'visible')
				counter.count()
			else {
				var f = function(e) {
					if (document.visibilityState !== 'visible')
						return
					document.removeEventListener('visibilitychange', f)
					counter.count()
				}
				document.addEventListener('visibilitychange', f)
			}

			if (!counter.no_events)
				counter.bind_events()
    })
})();