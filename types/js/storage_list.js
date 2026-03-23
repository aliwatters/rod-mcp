// storage_list.js — enumerates all keys/values from a named storage object.
// The placeholder %STORAGE_OBJ% is replaced at runtime with "localStorage" or "sessionStorage".
() => {
	const s = %STORAGE_OBJ%;
	const items = {};
	for (let i = 0; i < s.length; i++) {
		const key = s.key(i);
		items[key] = s.getItem(key);
	}
	return JSON.stringify(items, null, 2);
}
