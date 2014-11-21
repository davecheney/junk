package main

func init() {
	Register("chord", "chord", `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
      #tooltip {
        color: white;
        opacity: .9;
        background: #333;
        padding: 5px;
        border: 1px solid lightgrey;
        border-radius: 5px;
        position: absolute;
        z-index: 10;
        visibility: hidden;
        white-space: nowrap;
        pointer-events: none;
      }
      #circle circle {
        fill: none;
        pointer-events: all;
      }
      path.group {
        fill-opacity: .8;
      }
      path.chord {
        fill-opacity: .8;
        stroke: #000;
        stroke-width: .25px;
      }
      #circle:hover path.fade {
        display: none;
      }
    </style>
    <script type="text/javascript">
    var _gaq = _gaq || [];
    _gaq.push(['_setAccount', 'UA-45101494-1']);
    _gaq.push(['_setDomainName', 'delimited.io']);
    _gaq.push(['_trackPageview']);
    (function() {
        var ga = document.createElement('script'); ga.type = 'text/javascript'; ga.async = true;
        ga.src = ('https:' == document.location.protocol ? 'https://ssl' : 'http://www') + '.google-analytics.com/ga.js';
        var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(ga, s);
    })();
    </script>
  </head>
  <body>
    <div id="tooltip"></div>
    <script src="http://d3js.org/d3.v3.min.js"></script>
    <script src="//cdnjs.cloudflare.com/ajax/libs/underscore.js/1.7.0/underscore-min.js"></script>
<script>
//*******************************************************************
//  CHORD MAPPER 
//*******************************************************************
function chordMpr (data) {
  var mpr = {}, mmap = {}, n = 0,
      matrix = [], filter, accessor;

  mpr.setFilter = function (fun) {
    filter = fun;
    return this;
  },
  mpr.setAccessor = function (fun) {
    accessor = fun;
    return this;
  },
  mpr.getMatrix = function () {
    matrix = [];
    _.each(mmap, function (a) {
      if (!matrix[a.id]) matrix[a.id] = [];
      _.each(mmap, function (b) {
       var recs = _.filter(data, function (row) {
          return filter(row, a, b);
        })
        matrix[a.id][b.id] = accessor(recs, a, b);
      });
    });
    return matrix;
  },
  mpr.getMap = function () {
    return mmap;
  },
  mpr.printMatrix = function () {
    _.each(matrix, function (elem) {
      console.log(elem);
    })
  },
  mpr.addToMap = function (value, info) {
    if (!mmap[value]) {
      mmap[value] = { name: value, id: n++, data: info }
    }
  },
  mpr.addValuesToMap = function (varName, info) {
    var values = _.uniq(_.pluck(data, varName));
    _.map(values, function (v) {
      if (!mmap[v]) {
        mmap[v] = { name: v, id: n++, data: info }
      }
    });
    return this;
  }
  return mpr;
}
//*******************************************************************
//  CHORD READER
//*******************************************************************
function chordRdr (matrix, mmap) {
  return function (d) {
    var i,j,s,t,g,m = {};
    if (d.source) {
      i = d.source.index; j = d.target.index;
      s = _.where(mmap, {id: i });
      t = _.where(mmap, {id: j });
      m.sname = s[0].name;
      m.sdata = d.source.value;
      m.svalue = +d.source.value;
      m.stotal = _.reduce(matrix[i], function (k, n) { return k + n }, 0);
      m.tname = t[0].name;
      m.tdata = d.target.value;
      m.tvalue = +d.target.value;
      m.ttotal = _.reduce(matrix[j], function (k, n) { return k + n }, 0);
    } else {
      g = _.where(mmap, {id: d.index });
      m.gname = g[0].name;
      m.gdata = g[0].data;
      m.gvalue = d.value;
    }
    m.mtotal = _.reduce(matrix, function (m1, n1) { 
      return m1 + _.reduce(n1, function (m2, n2) { return m2 + n2}, 0);
    }, 0);
    return m;
  }
}
    </script>
    <script>
      //*******************************************************************
      //  CREATE MATRIX AND MAP
      //*******************************************************************
      d3.json('/cc/{{.package}}', function (error, data) {
        var mpr = chordMpr(data);

        _.each(data, function (elem) {
          mpr.addToMap(elem.name)
        })

        mpr.setFilter(function (row, a, b) {
            return (row.name === a.name)
          })
          .setAccessor(function (recs, a, b) {
            if (!recs[0]) return 0;
            var n = 0;
            _.each(recs, function (r) {
              _.each(r.imports, function (i) {
                if (i === b.name) n++;
              });
            });
            return n;
          });
        drawChords(mpr.getMatrix(), mpr.getMap());
      });

      //*******************************************************************
      //  DRAW THE CHORD DIAGRAM
      //*******************************************************************
      function drawChords (matrix, mmap) {
        var w = 980, h = 800, r1 = h / 2, r0 = r1 - 100;

        var fill = d3.scale.ordinal()
            .range(['#c7b570','#c6cdc7','#335c64','#768935','#507282','#5c4a56','#aa7455','#574109','#837722','#73342d','#0a5564','#9c8f57','#7895a4','#4a5456','#b0a690','#0a3542',]);

        var chord = d3.layout.chord()
            .padding(.04)
            .sortSubgroups(d3.descending)
            .sortChords(d3.descending);

        var arc = d3.svg.arc()
            .innerRadius(r0)
            .outerRadius(r0 + 20);

        var svg = d3.select("body").append("svg:svg")
            .attr("width", w)
            .attr("height", h)
          .append("svg:g")
            .attr("id", "circle")
            .attr("transform", "translate(" + w / 2 + "," + h / 2 + ")");

            svg.append("circle")
                .attr("r", r0 + 20);

        var rdr = chordRdr(matrix, mmap);
        chord.matrix(matrix);

        var g = svg.selectAll("g.group")
            .data(chord.groups())
          .enter().append("svg:g")
            .attr("class", "group")
            .on("mouseover", mouseover)
            .on("mouseout", function (d) { d3.select("#tooltip").style("visibility", "hidden") });

        g.append("svg:path")
            .style("stroke", "black")
            .style("fill", function(d) { return fill(rdr(d).gname); })
            .attr("d", arc);

        g.append("svg:text")
            .each(function(d) { d.angle = (d.startAngle + d.endAngle) / 2; })
            .attr("dy", ".35em")
            .style("font-family", "helvetica, arial, sans-serif")
            .style("font-size", "10px")
            .attr("text-anchor", function(d) { return d.angle > Math.PI ? "end" : null; })
            .attr("transform", function(d) {
              return "rotate(" + (d.angle * 180 / Math.PI - 90) + ")"
                  + "translate(" + (r0 + 26) + ")"
                  + (d.angle > Math.PI ? "rotate(180)" : "");
            })
            .text(function(d) { return rdr(d).gname; });

          var chordPaths = svg.selectAll("path.chord")
                .data(chord.chords())
              .enter().append("svg:path")
                .attr("class", "chord")
                .style("stroke", function(d) { return d3.rgb(fill(rdr(d).sname)).darker(); })
                .style("fill", function(d) { return fill(rdr(d).sname); })
                .attr("d", d3.svg.chord().radius(r0))
                .on("mouseover", function (d) {
                  d3.select("#tooltip")
                    .style("visibility", "visible")
                    .html(chordTip(rdr(d)))
                    .style("top", function () { return (d3.event.pageY - 170)+"px"})
                    .style("left", function () { return (d3.event.pageX - 100)+"px";})
                })
                .on("mouseout", function (d) { d3.select("#tooltip").style("visibility", "hidden") });

          function chordTip (d) {
            var p = d3.format(".1%"), q = d3.format(",.2r")
            return "Chord Info:<br/>"
              +  d.sname + " &#x2192; " + d.tname
              + ": " + q(d.svalue) + "<br/>"
              + p(d.svalue/d.stotal) + " of " + d.sname + "'s Total (" + q(d.stotal) + ")<br/>"
              + p(d.svalue/d.mtotal) + " of Matrix Total (" + q(d.mtotal) + ")";
          }

          function groupTip (d) {
            var p = d3.format(".1%"), q = d3.format(",.2r")
            return "Group Info:<br/>"
                + d.gname + " : " + q(d.gvalue) + "<br/>"
                + p(d.gvalue/d.mtotal) + " of Matrix Total (" + q(d.mtotal) + ")"
          }

          function mouseover(d, i) {
            d3.select("#tooltip")
              .style("visibility", "visible")
              .html(groupTip(rdr(d)))
              .style("top", function () { return (d3.event.pageY - 80)+"px"})
              .style("left", function () { return (d3.event.pageX - 130)+"px";})

            chordPaths.classed("fade", function(p) {
              return p.source.index != i
                  && p.target.index != i;
            });
          }
      }
    </script>
  </body>
</html>`)
}
