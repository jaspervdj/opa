package formatty

import future.keywords.every

testy {
	every k in input.numbers {
		k % 2 == 0
	}
	count(input.numbers) % 2 == 0
}

funky {
	foo := {
		"some_key": "some_value",
	}

	count(foo) == 1
}

wonky {
	foo := object.get(
	    input,
		"some_key",
		"some_value",
	)
	count(foo) == 1
}
