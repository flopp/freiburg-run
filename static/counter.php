<?php
header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: no-store, no-cache, must-revalidate');

$now = new DateTimeImmutable('now', new DateTimeZone('Europe/Berlin'));
$periods = [
    'day' => $now->format('Y-m-d'),
    'month' => $now->format('Y-m'),
    'year' => $now->format('Y'),
];

$counter = fopen(__DIR__ . '/counter.txt', 'c+');
if ($counter === false) {
    error_log('Unable to open visitor counter file.');
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

if (!flock($counter, LOCK_EX)) {
    error_log('Unable to lock visitor counter file.');
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

$contents = stream_get_contents($counter);
$value = trim($contents === false ? '' : $contents);
if ($contents === false) {
    error_log('Visitor counter file contains invalid data.');
    flock($counter, LOCK_UN);
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

if ($value === '') {
    $state = ['total' => 0, 'periods' => []];
} elseif (ctype_digit($value)) {
    $state = ['total' => (int) $value, 'periods' => []];
} else {
    $state = json_decode($value, true);
    if (
        !is_array($state)
        || !isset($state['total'], $state['periods'])
        || !is_int($state['total'])
        || $state['total'] < 0
        || !is_array($state['periods'])
    ) {
        error_log('Visitor counter file contains invalid data.');
        flock($counter, LOCK_UN);
        fclose($counter);
        http_response_code(500);
        echo json_encode(['error' => 'Counter unavailable']);
        exit;
    }
}

$counts = [];
foreach ($periods as $period => $key) {
    if (!array_key_exists($period, $state['periods'])) {
        $current = ['key' => $key, 'count' => 0];
    } else {
        $current = $state['periods'][$period];
        if (
            !is_array($current)
            || !isset($current['key'], $current['count'])
            || !is_string($current['key'])
            || !is_int($current['count'])
            || $current['count'] < 0
        ) {
            error_log('Visitor counter file contains invalid period data.');
            flock($counter, LOCK_UN);
            fclose($counter);
            http_response_code(500);
            echo json_encode(['error' => 'Counter unavailable']);
            exit;
        }
    }

    if ($current['key'] !== $key) {
        $current['count'] = 0;
    }

    if ($current['count'] === PHP_INT_MAX) {
        error_log('Visitor period count has reached the maximum value.');
        flock($counter, LOCK_UN);
        fclose($counter);
        http_response_code(500);
        echo json_encode(['error' => 'Counter unavailable']);
        exit;
    }

    $counts[$period] = $current['count'] + 1;
    $state['periods'][$period] = ['key' => $key, 'count' => $counts[$period]];
}

if ($state['total'] === PHP_INT_MAX) {
    error_log('Visitor total count has reached the maximum value.');
    flock($counter, LOCK_UN);
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}
$state['total']++;
$serialized = json_encode($state);
if ($serialized === false) {
    error_log('Unable to encode visitor counter data.');
    flock($counter, LOCK_UN);
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

if (!ftruncate($counter, 0) || !rewind($counter)) {
    error_log('Unable to update visitor counter file.');
    flock($counter, LOCK_UN);
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

if (fwrite($counter, $serialized) !== strlen($serialized) || !fflush($counter)) {
    error_log('Unable to write visitor counter file.');
    flock($counter, LOCK_UN);
    fclose($counter);
    http_response_code(500);
    echo json_encode(['error' => 'Counter unavailable']);
    exit;
}

flock($counter, LOCK_UN);
fclose($counter);
echo json_encode([
    'daily' => $counts['day'],
    'monthly' => $counts['month'],
    'yearly' => $counts['year'],
    'total' => $state['total'],
]);
