export const sendRequest = (input: RequestInfo | URL, init: RequestInit) =>
  fetch(input, init);
