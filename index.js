exports.handler = async (event, context) => {
  const response = {
    statusCode: 200,
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
    },
    body: "<p>Hello world!</p>",
  };

  return response;
};

