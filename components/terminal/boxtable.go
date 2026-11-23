// Code generated from the Unicode character names; DO NOT EDIT.

package terminal

// boxLines are the lines of the box drawing characters U+2500-U+257F:
// up, right, down and left, each 0 (none), 1 (light), 2 (heavy) or
// 3 (double). Dashed lines are drawn solid and arcs as corners.
var boxLines = map[rune][4]uint8{
	0x2500: {0, 1, 0, 1}, // box drawings light horizontal
	0x2501: {0, 2, 0, 2}, // box drawings heavy horizontal
	0x2502: {1, 0, 1, 0}, // box drawings light vertical
	0x2503: {2, 0, 2, 0}, // box drawings heavy vertical
	0x2504: {0, 1, 0, 1}, // box drawings light triple dash horizontal
	0x2505: {0, 2, 0, 2}, // box drawings heavy triple dash horizontal
	0x2506: {1, 0, 1, 0}, // box drawings light triple dash vertical
	0x2507: {2, 0, 2, 0}, // box drawings heavy triple dash vertical
	0x2508: {0, 1, 0, 1}, // box drawings light quadruple dash horizontal
	0x2509: {0, 2, 0, 2}, // box drawings heavy quadruple dash horizontal
	0x250A: {1, 0, 1, 0}, // box drawings light quadruple dash vertical
	0x250B: {2, 0, 2, 0}, // box drawings heavy quadruple dash vertical
	0x250C: {0, 1, 1, 0}, // box drawings light down and right
	0x250D: {0, 2, 1, 0}, // box drawings down light and right heavy
	0x250E: {0, 1, 2, 0}, // box drawings down heavy and right light
	0x250F: {0, 2, 2, 0}, // box drawings heavy down and right
	0x2510: {0, 0, 1, 1}, // box drawings light down and left
	0x2511: {0, 0, 1, 2}, // box drawings down light and left heavy
	0x2512: {0, 0, 2, 1}, // box drawings down heavy and left light
	0x2513: {0, 0, 2, 2}, // box drawings heavy down and left
	0x2514: {1, 1, 0, 0}, // box drawings light up and right
	0x2515: {1, 2, 0, 0}, // box drawings up light and right heavy
	0x2516: {2, 1, 0, 0}, // box drawings up heavy and right light
	0x2517: {2, 2, 0, 0}, // box drawings heavy up and right
	0x2518: {1, 0, 0, 1}, // box drawings light up and left
	0x2519: {1, 0, 0, 2}, // box drawings up light and left heavy
	0x251A: {2, 0, 0, 1}, // box drawings up heavy and left light
	0x251B: {2, 0, 0, 2}, // box drawings heavy up and left
	0x251C: {1, 1, 1, 0}, // box drawings light vertical and right
	0x251D: {1, 2, 1, 0}, // box drawings vertical light and right heavy
	0x251E: {2, 1, 1, 0}, // box drawings up heavy and right down light
	0x251F: {1, 1, 2, 0}, // box drawings down heavy and right up light
	0x2520: {2, 1, 2, 0}, // box drawings vertical heavy and right light
	0x2521: {2, 2, 1, 0}, // box drawings down light and right up heavy
	0x2522: {1, 2, 2, 0}, // box drawings up light and right down heavy
	0x2523: {2, 2, 2, 0}, // box drawings heavy vertical and right
	0x2524: {1, 0, 1, 1}, // box drawings light vertical and left
	0x2525: {1, 0, 1, 2}, // box drawings vertical light and left heavy
	0x2526: {2, 0, 1, 1}, // box drawings up heavy and left down light
	0x2527: {1, 0, 2, 1}, // box drawings down heavy and left up light
	0x2528: {2, 0, 2, 1}, // box drawings vertical heavy and left light
	0x2529: {2, 0, 1, 2}, // box drawings down light and left up heavy
	0x252A: {1, 0, 2, 2}, // box drawings up light and left down heavy
	0x252B: {2, 0, 2, 2}, // box drawings heavy vertical and left
	0x252C: {0, 1, 1, 1}, // box drawings light down and horizontal
	0x252D: {0, 1, 1, 2}, // box drawings left heavy and right down light
	0x252E: {0, 2, 1, 1}, // box drawings right heavy and left down light
	0x252F: {0, 2, 1, 2}, // box drawings down light and horizontal heavy
	0x2530: {0, 1, 2, 1}, // box drawings down heavy and horizontal light
	0x2531: {0, 1, 2, 2}, // box drawings right light and left down heavy
	0x2532: {0, 2, 2, 1}, // box drawings left light and right down heavy
	0x2533: {0, 2, 2, 2}, // box drawings heavy down and horizontal
	0x2534: {1, 1, 0, 1}, // box drawings light up and horizontal
	0x2535: {1, 1, 0, 2}, // box drawings left heavy and right up light
	0x2536: {1, 2, 0, 1}, // box drawings right heavy and left up light
	0x2537: {1, 2, 0, 2}, // box drawings up light and horizontal heavy
	0x2538: {2, 1, 0, 1}, // box drawings up heavy and horizontal light
	0x2539: {2, 1, 0, 2}, // box drawings right light and left up heavy
	0x253A: {2, 2, 0, 1}, // box drawings left light and right up heavy
	0x253B: {2, 2, 0, 2}, // box drawings heavy up and horizontal
	0x253C: {1, 1, 1, 1}, // box drawings light vertical and horizontal
	0x253D: {1, 1, 1, 2}, // box drawings left heavy and right vertical light
	0x253E: {1, 2, 1, 1}, // box drawings right heavy and left vertical light
	0x253F: {1, 2, 1, 2}, // box drawings vertical light and horizontal heavy
	0x2540: {2, 1, 1, 1}, // box drawings up heavy and down horizontal light
	0x2541: {1, 1, 2, 1}, // box drawings down heavy and up horizontal light
	0x2542: {2, 1, 2, 1}, // box drawings vertical heavy and horizontal light
	0x2543: {2, 1, 1, 2}, // box drawings left up heavy and right down light
	0x2544: {2, 2, 1, 1}, // box drawings right up heavy and left down light
	0x2545: {1, 1, 2, 2}, // box drawings left down heavy and right up light
	0x2546: {1, 2, 2, 1}, // box drawings right down heavy and left up light
	0x2547: {2, 2, 1, 2}, // box drawings down light and up horizontal heavy
	0x2548: {1, 2, 2, 2}, // box drawings up light and down horizontal heavy
	0x2549: {2, 1, 2, 2}, // box drawings right light and left vertical heavy
	0x254A: {2, 2, 2, 1}, // box drawings left light and right vertical heavy
	0x254B: {2, 2, 2, 2}, // box drawings heavy vertical and horizontal
	0x254C: {0, 1, 0, 1}, // box drawings light double dash horizontal
	0x254D: {0, 2, 0, 2}, // box drawings heavy double dash horizontal
	0x254E: {1, 0, 1, 0}, // box drawings light double dash vertical
	0x254F: {2, 0, 2, 0}, // box drawings heavy double dash vertical
	0x2550: {0, 3, 0, 3}, // box drawings double horizontal
	0x2551: {3, 0, 3, 0}, // box drawings double vertical
	0x2552: {0, 3, 1, 0}, // box drawings down single and right double
	0x2553: {0, 1, 3, 0}, // box drawings down double and right single
	0x2554: {0, 3, 3, 0}, // box drawings double down and right
	0x2555: {0, 0, 1, 3}, // box drawings down single and left double
	0x2556: {0, 0, 3, 1}, // box drawings down double and left single
	0x2557: {0, 0, 3, 3}, // box drawings double down and left
	0x2558: {1, 3, 0, 0}, // box drawings up single and right double
	0x2559: {3, 1, 0, 0}, // box drawings up double and right single
	0x255A: {3, 3, 0, 0}, // box drawings double up and right
	0x255B: {1, 0, 0, 3}, // box drawings up single and left double
	0x255C: {3, 0, 0, 1}, // box drawings up double and left single
	0x255D: {3, 0, 0, 3}, // box drawings double up and left
	0x255E: {1, 3, 1, 0}, // box drawings vertical single and right double
	0x255F: {3, 1, 3, 0}, // box drawings vertical double and right single
	0x2560: {3, 3, 3, 0}, // box drawings double vertical and right
	0x2561: {1, 0, 1, 3}, // box drawings vertical single and left double
	0x2562: {3, 0, 3, 1}, // box drawings vertical double and left single
	0x2563: {3, 0, 3, 3}, // box drawings double vertical and left
	0x2564: {0, 3, 1, 3}, // box drawings down single and horizontal double
	0x2565: {0, 1, 3, 1}, // box drawings down double and horizontal single
	0x2566: {0, 3, 3, 3}, // box drawings double down and horizontal
	0x2567: {1, 3, 0, 3}, // box drawings up single and horizontal double
	0x2568: {3, 1, 0, 1}, // box drawings up double and horizontal single
	0x2569: {3, 3, 0, 3}, // box drawings double up and horizontal
	0x256A: {1, 3, 1, 3}, // box drawings vertical single and horizontal double
	0x256B: {3, 1, 3, 1}, // box drawings vertical double and horizontal single
	0x256C: {3, 3, 3, 3}, // box drawings double vertical and horizontal
	0x256D: {0, 1, 1, 0}, // box drawings light arc down and right
	0x256E: {0, 0, 1, 1}, // box drawings light arc down and left
	0x256F: {1, 0, 0, 1}, // box drawings light arc up and left
	0x2570: {1, 1, 0, 0}, // box drawings light arc up and right
	0x2574: {0, 0, 0, 1}, // box drawings light left
	0x2575: {1, 0, 0, 0}, // box drawings light up
	0x2576: {0, 1, 0, 0}, // box drawings light right
	0x2577: {0, 0, 1, 0}, // box drawings light down
	0x2578: {0, 0, 0, 2}, // box drawings heavy left
	0x2579: {2, 0, 0, 0}, // box drawings heavy up
	0x257A: {0, 2, 0, 0}, // box drawings heavy right
	0x257B: {0, 0, 2, 0}, // box drawings heavy down
	0x257C: {0, 2, 0, 1}, // box drawings light left and heavy right
	0x257D: {1, 0, 2, 0}, // box drawings light up and heavy down
	0x257E: {0, 1, 0, 2}, // box drawings heavy left and light right
	0x257F: {2, 0, 1, 0}, // box drawings heavy up and light down
}
