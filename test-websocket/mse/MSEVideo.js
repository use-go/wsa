function judgeBrowser() {
    var ua = navigator.userAgent.toLowerCase();

    var browser_type = ua.match('edge');
    if (null != browser_type) {
        browser_type = browser_type[0];
    } else {
        browser_type = ua.match('chrome');
        if (null != browser_type) {
            browser_type = browser_type[0];
        } else {
            browser_type = ua.match('firefox');
            if (null != browser_type) {
                browser_type = browser_type[0];
            } else {
                browser_type = ua.match('trident');
                if (null != browser_type) {
                    browser_type = 'msie';
                } else {
                    browser_type = ua.match('safari');
                    if (null != browser_type) {
                        browser_type = browser_type[0];
                    }
                }
            }
        }
    }

    return browser_type;
}

function setupMSE(element_id, web_socket_url) {

    var video_element = null;
    var media_source;
    var video_source;
    var buffer_array = [];

    var web_socket;
    var interval_id;

    var last_end_time = 0;
    var equal_end_time_count = 0;
	var zero_buffer_length_count = 0;

    //var sequence_num = 1;
    //var time_stamp = new Uint8Array(8);

    setupVideo(element_id);
    setupWebSocket(web_socket_url);
    return { videoelement: video_element , closefunc : closehandle};
    
    // Create mediaSource and initialize video 
    function setupVideo(element_id) {
        //  Create the media source
        if (window.MediaSource) {
            media_source = new window.MediaSource();
        } else {
            console.log("mediasource or syntax not supported");
            return;
        }

        var url = URL.createObjectURL(media_source);
        var videoParent = document.getElementById(element_id);
		if (null == video_element) {
			video_element = document.createElement('video');
			video_element.autoplay = true;
			video_element.style.width = '100%';
			video_element.style.height = '100%';
			videoParent.appendChild(video_element);
		}
        video_element.src = url;

        // Wait for event that tells us our media source object is 
        // ready for a buffer to be added.
        media_source.addEventListener('sourceopen', sourceopenhandle, false);
    }
	
	function closehandle() {
		web_socket.onclose = closeWebsocketforCloseVideo;
		web_socket.close();
		video_element.remove();
	}

    function sourceopenhandle(evt) {
        video_source = media_source.addSourceBuffer('video/mp4; codecs="avc1.640029"');
        media_source.duration = Infinity;
        media_source.removeEventListener('sourceopen', sourceopenhandle, false);
        video_source.addEventListener('updateend', readFromBuffer, false);
    }

    function readFromBuffer() {
        // Return if the buffer is empty or has state updating
        if (buffer_array.length <= 0 || video_source.updating) {
            return;
        }
        try {
            var data = buffer_array.shift();
            var data2 = new Uint8Array(data);
            //processSequenceNum(data2);
            video_source.appendBuffer(data2);
        } catch (e) { }
    }

    //function processSequenceNum(data) {
    //    // compare with 'moof'
    //    if (data[4] == 109 && data[5] == 111 && data[6] == 111 && data[7] == 102) {
    //        // moof box
    //        var data_index = 8;
    //        // mfhd box
    //        var box_size = data[data_index + 3];
    //        data_index += box_size;
    //        // sequence number is the last 4 bytes in mfhd box
    //        var packet_sequence_num = data[20] << 24 | data[21] << 16 | data[22] << 8 | data[23];
    //        if (packet_sequence_num != sequence_num) {
    //        data[data_index - 4] = (sequence_num & 0xff000000) >> 24;
    //        data[data_index - 3] = (sequence_num & 0x00ff0000) >> 16;
    //        data[data_index - 2] = (sequence_num & 0x0000ff00) >> 8;
    //        data[data_index - 1] = sequence_num & 0x000000ff;
    //        }
    //        ++sequence_num;

    //        // traf box
    //        data_index += 8;
    //        // tfhd box
    //        box_size = data[data_index + 3];
    //        data_index += box_size;
    //        // tfdt box
    //        data_index += 12;
    //        // base media decode time is 64bit in tfdt box
    //        //console.log(time_stamp[0]<<56 | time_stamp[1]<<48 | time_stamp[2]<<40 | time_stamp[3]<<32 | time_stamp[4]<<24 | time_stamp[5]<<16 | time_stamp[6]<<8 | time_stamp[7]);
    //        for (var i = 0; i < 8; ++i) {
    //            data[data_index + i] = time_stamp[i];
    //        }
    //        var digit = 7;
    //        var high = 2000;
    //        var low;
    //        while (high != 0 && digit >= 0) {
    //            low = time_stamp[digit] + high;
    //            high = (low & 0xffffff00) >> 8;
    //            time_stamp[digit] = low & 0xff;
    //            --digit;
    //        }
    //    }
    //}

    function setupWebSocket(web_socket_url) {
        var sub_protocol = 'lws-video';
        web_socket = new WebSocket(web_socket_url, sub_protocol);
        web_socket.binaryType = 'arraybuffer';

        web_socket.onopen = function (evt) {
            interval_id = setInterval(timeInterval, 2000);
        }

        web_socket.onmessage = function (message) {
            processMP4Box(message.data);
        }

        web_socket.onclose = function (evt) {
            resetWebsocket();
            resetVideo();
            setupVideo(element_id);
            setupWebSocket(web_socket_url);
        }
    }

    function timeInterval(evt) {
        if (video_source.buffered.length > 0) {
			zero_buffer_length_count = 0;
            var start_time = video_source.buffered.start(0);
            var end_time = video_source.buffered.end(0);

            // Judge no more data in video stream
            if (last_end_time == end_time) {
                ++equal_end_time_count;
                if (equal_end_time_count > 4) {
					var d = new Date();
                    console.log(d.getDate() + ' ' + d.getHours() + ':' + d.getMinutes() + ':' + d.getSeconds() + ' end_time stop');
                    web_socket.close();
                }
            } else {
                equal_end_time_count = 0;
            }

            last_end_time = end_time;

            // Judge delay
            var diff = end_time - video_element.currentTime;
            if (diff < 0 || diff > 1.5) {
				var d = new Date();
				console.log(d.getDate() + ' ' + d.getHours() + ':' + d.getMinutes() + ':' + d.getSeconds() + ' delay');
                web_socket.close();
            }

            // Clear video cache
            if (video_element.currentTime - start_time > 30) {
                if (!video_source.updating) {
                    video_source.remove(start_time, start_time + 25);
                }
            }
        } else {
			++zero_buffer_length_count;
			if (zero_buffer_length_count > 10) {
				var d = new Date();
				console.log(d.getDate() + ' ' + d.getHours() + ':' + d.getMinutes() + ':' + d.getSeconds() + ' no data');
                web_socket.close();
			}
		}
    }

    function processMP4Box(mp4_box) {
        buffer_array.push(mp4_box);
        readFromBuffer();
    }

    function resetWebsocket() {
        clearInterval(interval_id);
        last_end_time = 0;
        equal_end_time_count = 0;
		zero_buffer_length_count = 0;
    }

    function resetVideo() {
        video_source.removeEventListener('updateend', readFromBuffer, false);
        video_source.abort();
        media_source.removeSourceBuffer(video_source);
		video_element.src = '';
        video_source = null;
        media_source = null;
        buffer_array = [];
    }

    function closeWebsocketforCloseVideo() {
        resetWebsocket();
        resetVideo();
    }
}

function closeMSE(mse_object) {
    mse_object.closefunc();
}