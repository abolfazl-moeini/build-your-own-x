export default (req, res) => {
    const data = req.body;
    res.send(`Got data: ${JSON.stringify(data)}`);
};